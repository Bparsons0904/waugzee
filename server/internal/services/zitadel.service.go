package services

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCDiscovery represents OIDC discovery document
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKS_URI              string `json:"jwks_uri"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSet represents a set of JSON Web Keys
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// ZitadelService handles OIDC authentication and user management
type ZitadelService struct {
	config       config.Config
	log          logger.Logger
	httpClient   *http.Client
	issuer       string
	clientID     string
	clientSecret string
	privateKey   *rsa.PrivateKey
	keyID        string
	clientIDM2M  string

	// OIDC discovery and JWK caching
	discovery     *OIDCDiscovery
	jwks          *JWKSet
	discoveryMux  sync.RWMutex
	jwksMux       sync.RWMutex
	discoveryTime time.Time
	jwksTime      time.Time
	cacheTTL      time.Duration
}

// TokenInfo represents validated token information
type TokenInfo struct {
	UserID          string
	Email           string
	Name            string
	GivenName       string
	FamilyName      string
	PreferredName   string
	EmailVerified   bool
	Roles           []string
	ProjectID       string
	Nonce           string
	Valid           bool
}


func NewZitadelService(cfg config.Config) (*ZitadelService, error) {
	log := logger.New("ZitadelService")

	// Require Zitadel configuration
	if cfg.ZitadelInstanceURL == "" || cfg.ZitadelClientID == "" {
		return nil, log.Err("Zitadel configuration required but not provided", 
			fmt.Errorf("missing ZitadelInstanceURL or ZitadelClientID"))
	}

	// Parse private key if available (for machine-to-machine authentication)
	var privateKey *rsa.PrivateKey
	if cfg.ZitadelPrivateKey != "" {
		// Decode base64 private key
		keyBytes, err := base64.StdEncoding.DecodeString(cfg.ZitadelPrivateKey)
		if err != nil {
			return nil, log.Err("failed to decode Zitadel private key", err)
		}

		// Parse RSA private key
		privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
		if err != nil {
			return nil, log.Err("failed to parse Zitadel private key", err)
		}
		log.Info("Zitadel private key loaded successfully")
	}

	// Create HTTP client with reasonable timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	service := &ZitadelService{
		config:       cfg,
		log:          log,
		httpClient:   httpClient,
		issuer:       cfg.ZitadelInstanceURL,
		clientID:     cfg.ZitadelClientID,
		clientSecret: cfg.ZitadelClientSecret,
		privateKey:   privateKey,
		keyID:        cfg.ZitadelKeyID,
		clientIDM2M:  cfg.ZitadelClientIDM2M,
		cacheTTL:     15 * time.Minute, // Cache OIDC discovery and JWKS for 15 minutes
	}

	log.Info("Zitadel service initialized successfully",
		"issuer", cfg.ZitadelInstanceURL,
		"hasPrivateKey", privateKey != nil)
	return service, nil
}

// ValidateIDToken validates an OIDC ID token with proper JWT signature verification
func (zs *ZitadelService) ValidateIDToken(ctx context.Context, idToken string) (*TokenInfo, error) {
	log := zs.log.Function("ValidateIDToken")

	// Parse JWT token with signature verification
	token, err := jwt.ParseWithClaims(
		idToken,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			// Validate algorithm
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// Get key ID from header
			kidHeader, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid 'kid' in JWT header")
			}

			// Get public key for verification
			publicKey, err := zs.getPublicKeyForToken(ctx, kidHeader)
			if err != nil {
				return nil, fmt.Errorf("failed to get public key: %w", err)
			}

			return publicKey, nil
		},
	)
	if err != nil {
		return &TokenInfo{Valid: false}, log.Err("JWT signature verification failed", err)
	}

	if !token.Valid {
		return &TokenInfo{Valid: false}, log.Err("JWT token is invalid", nil)
	}

	// Extract registered claims
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return &TokenInfo{Valid: false}, log.Err("failed to extract JWT claims", nil)
	}

	// Verify issuer
	expectedIssuer := strings.TrimSuffix(zs.issuer, "/")
	if claims.Issuer != expectedIssuer {
		return &TokenInfo{Valid: false}, log.Err(
			"invalid issuer",
			fmt.Errorf("expected: %s, got: %s", expectedIssuer, claims.Issuer),
		)
	}

	// Verify audience (client ID)
	audienceValid := slices.Contains(claims.Audience, zs.clientID)

	if !audienceValid {
		return &TokenInfo{Valid: false}, log.Err(
			"invalid audience",
			fmt.Errorf(
				"expected client ID %s not found in audience %v",
				zs.clientID,
				claims.Audience,
			),
		)
	}

	// Parse custom claims for user info (need to parse the original token again for custom claims)
	var customClaims struct {
		jwt.RegisteredClaims
		Email           string   `json:"email"`
		Name            string   `json:"name"`
		GivenName       string   `json:"given_name"`
		FamilyName      string   `json:"family_name"`
		PreferredName   string   `json:"preferred_username"`
		EmailVerified   bool     `json:"email_verified"`
		Nonce           string   `json:"nonce"`
		// Zitadel specific claims
		Roles           []string `json:"urn:zitadel:iam:org:project:roles"`
		ProjectID       string   `json:"urn:zitadel:iam:org:project:id"`
		// Alternative project ID location
		AzpProjectID    string   `json:"azp"`
	}

	// Parse again with custom claims struct
	_, err = jwt.ParseWithClaims(
		idToken,
		&customClaims,
		func(token *jwt.Token) (any, error) {
			kidHeader, _ := token.Header["kid"].(string)
			return zs.getPublicKeyForToken(ctx, kidHeader)
		},
	)
	if err != nil {
		log.Warn("failed to parse custom claims, using basic claims", "error", err)
		// Continue with basic claims if custom parsing fails
	}

	log.Info("ID token validation successful",
		"sub", claims.Subject,
		"email", customClaims.Email,
		"exp", claims.ExpiresAt.Time,
		"iss", claims.Issuer)

	// Build display name if 'name' is missing but we have given/family names
	displayName := customClaims.Name
	if displayName == "" && (customClaims.GivenName != "" || customClaims.FamilyName != "") {
		displayName = strings.TrimSpace(customClaims.GivenName + " " + customClaims.FamilyName)
	}

	// Extract project ID (try multiple claim locations)
	projectID := customClaims.ProjectID
	if projectID == "" {
		projectID = customClaims.AzpProjectID
	}

	return &TokenInfo{
		UserID:        claims.Subject,
		Email:         customClaims.Email,
		Name:          displayName,
		GivenName:     customClaims.GivenName,
		FamilyName:    customClaims.FamilyName,
		PreferredName: customClaims.PreferredName,
		EmailVerified: customClaims.EmailVerified,
		Roles:         customClaims.Roles,
		ProjectID:     projectID,
		Nonce:         customClaims.Nonce,
		Valid:         true,
	}, nil
}

// ValidateToken validates an access token using OIDC introspection
func (zs *ZitadelService) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {

	log := zs.log.Function("ValidateToken")

	// Create introspection request
	introspectURL := strings.TrimSuffix(zs.issuer, "/") + "/oauth/v2/introspect"

	// Prepare form data
	data := url.Values{}
	data.Set("token", token)

	// Use JWT assertion for M2M authentication if private key is available
	if zs.privateKey != nil {
		jwtAssertion, err := zs.generateJWTAssertion()
		if err != nil {
			return nil, log.Err("failed to generate JWT assertion", err)
		}

		// Use client_assertion method for JWT authentication
		data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
		data.Set("client_assertion", jwtAssertion)
	} else {
		// Fallback to client_id/client_secret for web client
		data.Set("client_id", zs.clientID)
		if zs.clientSecret != "" {
			data.Set("client_secret", zs.clientSecret)
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		introspectURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, log.Err("failed to create introspection request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return nil, log.Err("failed to make introspection request", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Info("failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, log.Err("introspection request failed",
			fmt.Errorf("status code: %d", resp.StatusCode),
			"responseBody", string(body))
	}

	var introspectResp struct {
		Active        bool     `json:"active"`
		Sub           string   `json:"sub"`
		Email         string   `json:"email"`
		Name          string   `json:"name"`
		GivenName     string   `json:"given_name"`
		FamilyName    string   `json:"family_name"`
		PreferredName string   `json:"preferred_username"`
		EmailVerified bool     `json:"email_verified"`
		// Zitadel specific claims
		Roles         []string `json:"urn:zitadel:iam:org:project:roles"`
		ProjectID     string   `json:"urn:zitadel:iam:org:project:id"`
		AzpProjectID  string   `json:"azp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&introspectResp); err != nil {
		return nil, log.Err("failed to decode introspection response", err)
	}

	if !introspectResp.Active {
		return &TokenInfo{Valid: false}, nil
	}

	// Build display name if not provided but we have given/family names
	displayName := introspectResp.Name
	if displayName == "" && (introspectResp.GivenName != "" || introspectResp.FamilyName != "") {
		displayName = strings.TrimSpace(introspectResp.GivenName + " " + introspectResp.FamilyName)
	}

	// Extract project ID (try multiple claim locations)
	projectID := introspectResp.ProjectID
	if projectID == "" {
		projectID = introspectResp.AzpProjectID
	}

	// Extract user information from introspection response
	return &TokenInfo{
		UserID:        introspectResp.Sub,
		Email:         introspectResp.Email,
		Name:          displayName,
		GivenName:     introspectResp.GivenName,
		FamilyName:    introspectResp.FamilyName,
		PreferredName: introspectResp.PreferredName,
		EmailVerified: introspectResp.EmailVerified,
		Roles:         introspectResp.Roles,
		ProjectID:     projectID,
		Valid:         true,
	}, nil
}





// getOIDCDiscovery fetches and caches the OIDC discovery document
func (zs *ZitadelService) getOIDCDiscovery(ctx context.Context) (*OIDCDiscovery, error) {

	log := zs.log.Function("getOIDCDiscovery")

	// Check cache first
	zs.discoveryMux.RLock()
	if zs.discovery != nil && time.Since(zs.discoveryTime) < zs.cacheTTL {
		discovery := zs.discovery
		zs.discoveryMux.RUnlock()
		return discovery, nil
	}
	zs.discoveryMux.RUnlock()

	// Fetch from OIDC discovery endpoint
	discoveryURL := strings.TrimSuffix(zs.issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return nil, log.Err("failed to create discovery request", err)
	}

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return nil, log.Err("failed to fetch OIDC discovery", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Info("failed to close discovery response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, log.Err("OIDC discovery request failed",
			fmt.Errorf("status code: %d", resp.StatusCode))
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, log.Err("failed to decode OIDC discovery", err)
	}

	// Validate discovery document
	if discovery.Issuer != strings.TrimSuffix(zs.issuer, "/") {
		return nil, log.Err("invalid issuer in discovery document",
			fmt.Errorf("expected: %s, got: %s", zs.issuer, discovery.Issuer))
	}

	if discovery.JWKS_URI == "" {
		return nil, log.Err("missing JWKS URI in discovery document", nil)
	}

	// Cache the discovery document
	zs.discoveryMux.Lock()
	zs.discovery = &discovery
	zs.discoveryTime = time.Now()
	zs.discoveryMux.Unlock()

	log.Info("OIDC discovery fetched successfully", "jwks_uri", discovery.JWKS_URI)
	return &discovery, nil
}

// getJWKS fetches and caches the JSON Web Key Set
func (zs *ZitadelService) getJWKS(ctx context.Context) (*JWKSet, error) {

	log := zs.log.Function("getJWKS")

	// Check cache first
	zs.jwksMux.RLock()
	if zs.jwks != nil && time.Since(zs.jwksTime) < zs.cacheTTL {
		jwks := zs.jwks
		zs.jwksMux.RUnlock()
		return jwks, nil
	}
	zs.jwksMux.RUnlock()

	// Get OIDC discovery to find JWKS URI
	discovery, err := zs.getOIDCDiscovery(ctx)
	if err != nil {
		return nil, log.Err("failed to get OIDC discovery for JWKS", err)
	}

	// Fetch JWKS
	req, err := http.NewRequestWithContext(ctx, "GET", discovery.JWKS_URI, nil)
	if err != nil {
		return nil, log.Err("failed to create JWKS request", err)
	}

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return nil, log.Err("failed to fetch JWKS", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Info("failed to close JWKS response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, log.Err("JWKS request failed",
			fmt.Errorf("status code: %d", resp.StatusCode))
	}

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, log.Err("failed to decode JWKS", err)
	}

	if len(jwks.Keys) == 0 {
		return nil, log.Err("JWKS contains no keys", nil)
	}

	// Cache the JWKS
	zs.jwksMux.Lock()
	zs.jwks = &jwks
	zs.jwksTime = time.Now()
	zs.jwksMux.Unlock()

	log.Info("JWKS fetched successfully", "keys_count", len(jwks.Keys))
	return &jwks, nil
}

// getPublicKeyForToken retrieves the public key for JWT verification based on kid header
func (zs *ZitadelService) getPublicKeyForToken(
	ctx context.Context,
	kidHeader string,
) (*rsa.PublicKey, error) {

	log := zs.log.Function("getPublicKeyForToken")

	// Get JWKS
	jwks, err := zs.getJWKS(ctx)
	if err != nil {
		return nil, log.Err("failed to get JWKS", err)
	}

	// Find matching key by kid
	var targetJWK *JWK
	for _, jwk := range jwks.Keys {
		if jwk.Kid == kidHeader {
			targetJWK = &jwk
			break
		}
	}

	if targetJWK == nil {
		return nil, log.Err("no matching key found",
			fmt.Errorf("kid: %s not found in JWKS", kidHeader))
	}

	// Validate key type
	if targetJWK.Kty != "RSA" {
		return nil, log.Err("unsupported key type",
			fmt.Errorf("expected RSA, got: %s", targetJWK.Kty))
	}

	// Decode RSA public key components
	nBytes, err := base64.RawURLEncoding.DecodeString(targetJWK.N)
	if err != nil {
		return nil, log.Err("failed to decode RSA modulus (n)", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(targetJWK.E)
	if err != nil {
		return nil, log.Err("failed to decode RSA exponent (e)", err)
	}

	// Convert to big.Int
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// Validate RSA exponent fits in int (prevent overflow on 32-bit systems)
	if !e.IsInt64() || e.Int64() > int64(^uint(0)>>1) {
		return nil, log.Err("RSA exponent too large",
			fmt.Errorf("exponent: %s", e.String()))
	}

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	log.Debug("public key retrieved successfully", "kid", kidHeader, "keyType", targetJWK.Kty)
	return publicKey, nil
}


// RevokeToken revokes an access or refresh token with Zitadel
func (zs *ZitadelService) RevokeToken(ctx context.Context, token string, tokenType string) error {

	log := zs.log.Function("RevokeToken")

	// Get OIDC discovery to find revocation endpoint
	discovery, err := zs.getOIDCDiscovery(ctx)
	if err != nil {
		return log.Err("failed to get OIDC discovery for token revocation", err)
	}

	if discovery.RevocationEndpoint == "" {
		return log.Err("revocation endpoint not available", fmt.Errorf("revocation_endpoint not found in OIDC discovery"))
	}

	// Prepare form data for revocation request
	data := url.Values{}
	data.Set("token", token)
	if tokenType != "" {
		data.Set("token_type_hint", tokenType)
	}

	// Use JWT assertion for M2M authentication if private key is available
	if zs.privateKey != nil {
		jwtAssertion, err := zs.generateJWTAssertion()
		if err != nil {
			return log.Err("failed to generate JWT assertion for revocation", err)
		}

		// Use client_assertion method for JWT authentication
		data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
		data.Set("client_assertion", jwtAssertion)
	} else {
		// Fallback to client_id/client_secret for web client
		data.Set("client_id", zs.clientID)
		if zs.clientSecret != "" {
			data.Set("client_secret", zs.clientSecret)
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		discovery.RevocationEndpoint,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return log.Err("failed to create token revocation request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := zs.httpClient.Do(req)
	if err != nil {
		return log.Err("failed to make token revocation request", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Info("failed to close revocation response body", "error", closeErr)
		}
	}()

	// RFC 7009 states that revocation endpoint should return 200 for successful revocation
	// or invalid tokens (to prevent token scanning attacks)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return log.Err("token revocation request failed",
			fmt.Errorf("status code: %d", resp.StatusCode),
			"responseBody", string(body))
	}

	log.Info("token revocation successful", "tokenType", tokenType, "endpoint", discovery.RevocationEndpoint)
	return nil
}

// GetLogoutURL generates the OIDC logout URL using the end_session_endpoint
func (zs *ZitadelService) GetLogoutURL(ctx context.Context, idTokenHint, postLogoutRedirectURI, state string) (string, error) {

	log := zs.log.Function("GetLogoutURL")

	// Get OIDC discovery to find end session endpoint
	discovery, err := zs.getOIDCDiscovery(ctx)
	if err != nil {
		return "", log.Err("failed to get OIDC discovery for logout URL", err)
	}

	if discovery.EndSessionEndpoint == "" {
		return "", log.Err("end session endpoint not available", fmt.Errorf("end_session_endpoint not found in OIDC discovery"))
	}

	// Build logout URL with query parameters
	logoutURL, err := url.Parse(discovery.EndSessionEndpoint)
	if err != nil {
		return "", log.Err("failed to parse end session endpoint", err)
	}

	params := url.Values{}
	
	// Add ID token hint if provided (recommended for better UX)
	if idTokenHint != "" {
		params.Set("id_token_hint", idTokenHint)
	}

	// Add post logout redirect URI if provided
	if postLogoutRedirectURI != "" {
		params.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	// Add state parameter if provided (for maintaining application state across logout)
	if state != "" {
		params.Set("state", state)
	}

	logoutURL.RawQuery = params.Encode()

	log.Info("logout URL generated successfully", 
		"endpoint", discovery.EndSessionEndpoint,
		"hasIdToken", idTokenHint != "",
		"hasRedirectURI", postLogoutRedirectURI != "")

	return logoutURL.String(), nil
}

// ZitadelConfig represents the OIDC configuration for clients
type ZitadelConfig struct {
	Domain      string `json:"domain"`
	InstanceURL string `json:"instanceUrl"`
	ClientID    string `json:"clientId"`
}

// GetConfig returns the OIDC configuration for client consumption
func (zs *ZitadelService) GetConfig() ZitadelConfig {
	return ZitadelConfig{
		Domain:      strings.TrimPrefix(zs.issuer, "https://"),
		InstanceURL: zs.issuer,
		ClientID:    zs.clientID,
	}
}

// ValidateTokenWithFallback validates a token using JWT-first approach with introspection fallback
func (zs *ZitadelService) ValidateTokenWithFallback(ctx context.Context, token string) (*TokenInfo, string, error) {
	log := zs.log.Function("ValidateTokenWithFallback")

	// Validate token with Zitadel - try JWT first, fallback to introspection
	var tokenInfo *TokenInfo
	var err error
	var validationMethod string

	if isJWTToken(token) {
		// Try JWT validation first (local, fast)
		tokenInfo, err = zs.ValidateIDToken(ctx, token)
		validationMethod = "JWT"
		
		// If JWT validation fails due to token format/type, fallback to introspection
		if err != nil {
			log.Debug("JWT validation failed, falling back to introspection", "error", err.Error())
			tokenInfo, err = zs.ValidateToken(ctx, token)
			validationMethod = "introspection_fallback"
		}
	} else {
		// Not a JWT token, use introspection directly
		tokenInfo, err = zs.ValidateToken(ctx, token)
		validationMethod = "introspection"
	}

	if err != nil {
		log.Info("token validation failed", "method", validationMethod, "error", err.Error())
		return nil, "", fmt.Errorf("token validation failed: %w", err)
	}

	if !tokenInfo.Valid {
		log.Info("token is not active", "method", validationMethod)
		return nil, "", fmt.Errorf("token is not active")
	}

	log.Info(
		"token validated successfully",
		"method",
		validationMethod,
		"userID",
		tokenInfo.UserID,
		"email",
		tokenInfo.Email,
	)
	return tokenInfo, validationMethod, nil
}

// isJWTToken checks if a token has JWT structure (3 base64 segments separated by dots)
func isJWTToken(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 &&
		len(parts[0]) > 0 &&
		len(parts[1]) > 0 &&
		len(parts[2]) > 0
}

// generateJWTAssertion creates a JWT assertion for machine-to-machine authentication
func (zs *ZitadelService) generateJWTAssertion() (string, error) {
	if zs.privateKey == nil {
		return "", fmt.Errorf("private key not configured")
	}

	log := zs.log.Function("generateJWTAssertion")

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    zs.clientIDM2M, // Use machine-to-machine clientId
		Subject:   zs.clientIDM2M, // Use machine-to-machine clientId
		Audience:  []string{strings.TrimSuffix(zs.issuer, "/")},
		ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}

	// Create token with key ID in header
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = zs.keyID

	signedToken, err := token.SignedString(zs.privateKey)
	if err != nil {
		return "", log.Err("failed to sign JWT assertion", err)
	}

	log.Debug("JWT assertion generated successfully", "keyID", zs.keyID)
	return signedToken, nil
}

// Close cleans up the Zitadel service resources
func (zs *ZitadelService) Close() error {
	// No resources to clean up for HTTP client
	return nil
}
