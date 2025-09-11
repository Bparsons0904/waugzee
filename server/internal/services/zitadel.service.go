package services

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

// ZitadelService handles OIDC authentication and user management
type ZitadelService struct {
	config       config.Config
	log          logger.Logger
	httpClient   *http.Client
	issuer       string
	clientID     string
	clientSecret string
	apiID        string
	privateKey   *rsa.PrivateKey
	keyID        string
	clientIDM2M  string
	configured   bool
}

// TokenInfo represents validated token information
type TokenInfo struct {
	UserID    string
	Email     string
	Name      string
	Roles     []string
	ProjectID string
	Valid     bool
}

// TokenExchangeRequest represents the token exchange request
type TokenExchangeRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state,omitempty"`
}

// TokenExchangeResponse represents the token exchange response
type TokenExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

func NewZitadelService(cfg config.Config) (*ZitadelService, error) {
	log := logger.New("ZitadelService")

	// Skip initialization if Zitadel is not configured
	if cfg.ZitadelInstanceURL == "" || cfg.ZitadelClientID == "" {
		log.Info("Zitadel configuration not found, service will be disabled")
		return &ZitadelService{
			config:     cfg,
			log:        log,
			configured: false,
		}, nil
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
		apiID:        cfg.ZitadelAPIID,
		privateKey:   privateKey,
		keyID:        cfg.ZitadelKeyID,
		clientIDM2M:  cfg.ZitadelClientIDM2M,
		configured:   true,
	}

	log.Info("Zitadel service initialized successfully",
		"issuer", cfg.ZitadelInstanceURL,
		"hasPrivateKey", privateKey != nil,
		"apiID", cfg.ZitadelAPIID)
	return service, nil
}

// ValidateIDToken validates an OIDC ID token by decoding its claims
// Note: This is a basic implementation that decodes the JWT payload without signature verification
// For production use, you should verify the JWT signature using the OIDC discovery endpoint
func (zs *ZitadelService) ValidateIDToken(ctx context.Context, idToken string) (*TokenInfo, error) {
	if !zs.configured {
		return nil, fmt.Errorf("zitadel service not configured")
	}

	log := zs.log.Function("ValidateIDToken")

	// Split JWT token into parts
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, log.Err(
			"invalid JWT format",
			fmt.Errorf("expected 3 parts, got %d", len(parts)),
		)
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, log.Err("failed to decode JWT payload", err)
	}

	// Parse the claims
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
		Exp   int64  `json:"exp"`
		Iss   string `json:"iss"`
		Aud   any    `json:"aud"` // Can be string or []string
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, log.Err("failed to parse JWT claims", err)
	}

	// Basic validation
	now := time.Now().Unix()
	if claims.Exp < now {
		return &TokenInfo{
				Valid: false,
			}, log.Err(
				"token expired",
				fmt.Errorf("exp: %d, now: %d", claims.Exp, now),
			)
	}

	// Verify issuer
	expectedIssuer := strings.TrimSuffix(zs.issuer, "/")
	if claims.Iss != expectedIssuer {
		return &TokenInfo{
				Valid: false,
			}, log.Err(
				"invalid issuer",
				fmt.Errorf("expected: %s, got: %s", expectedIssuer, claims.Iss),
			)
	}

	// Verify audience (client ID) - can be string or array
	audienceValid := false
	switch aud := claims.Aud.(type) {
	case string:
		audienceValid = aud == zs.clientID
	case []any:
		for _, a := range aud {
			if str, ok := a.(string); ok && str == zs.clientID {
				audienceValid = true
				break
			}
		}
	}

	if !audienceValid {
		return &TokenInfo{
				Valid: false,
			}, log.Err(
				"invalid audience",
				fmt.Errorf("expected client ID %s not found in audience", zs.clientID),
			)
	}

	log.Info("ID token validation successful", "sub", claims.Sub, "email", claims.Email)

	return &TokenInfo{
		UserID:    claims.Sub,
		Email:     claims.Email,
		Name:      claims.Name,
		Roles:     []string{}, // Roles would need to be in custom claims
		ProjectID: "",
		Valid:     true,
	}, nil
}

// ValidateToken validates an access token using OIDC introspection
func (zs *ZitadelService) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	if !zs.configured {
		return nil, fmt.Errorf("zitadel service not configured")
	}

	log := zs.log.Function("ValidateToken")

	// Create introspection request
	introspectURL := strings.TrimSuffix(zs.issuer, "/") + "/oauth/v2/introspect"

	// Prepare form data
	data := url.Values{}
	data.Set("token", token)

	// Use JWT authentication if private key is available (machine-to-machine)
	if zs.privateKey != nil {
		jwtAssertion, err := zs.generateJWTAssertion()
		if err != nil {
			return nil, log.Err("failed to generate JWT assertion", err)
		}

		// Use client_assertion method for JWT authentication
		data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
		data.Set("client_assertion", jwtAssertion)
	} else if zs.clientSecret != "" {
		// Fallback to client credentials for confidential clients
		data.Set("client_id", zs.clientID)
		data.Set("client_secret", zs.clientSecret)
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
		Active bool   `json:"active"`
		Sub    string `json:"sub"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&introspectResp); err != nil {
		return nil, log.Err("failed to decode introspection response", err)
	}

	if !introspectResp.Active {
		return &TokenInfo{Valid: false}, nil
	}

	// Extract user information from introspection response
	return &TokenInfo{
		UserID:    introspectResp.Sub,
		Email:     introspectResp.Email,
		Name:      introspectResp.Name,
		Roles:     []string{}, // Roles would need to be extracted from custom claims
		ProjectID: "",         // Project ID not needed for OIDC flow
		Valid:     true,
	}, nil
}

// GetUserInfo retrieves detailed user information by user ID
func (zs *ZitadelService) GetUserInfo(ctx context.Context, userID string) (map[string]any, error) {
	if !zs.configured {
		return nil, fmt.Errorf("zitadel service not configured")
	}

	log := zs.log.Function("GetUserInfo")

	// This is a placeholder implementation
	// In a real implementation, you would call the Zitadel Management API
	log.Info("user info requested", "userID", userID)

	return map[string]any{
		"id":   userID,
		"note": "User info retrieval would require Zitadel Management API integration",
	}, nil
}

// IsConfigured returns true if Zitadel is properly configured
func (zs *ZitadelService) IsConfigured() bool {
	return zs.configured
}

// GetAuthorizationURL generates authorization URL for OIDC flow
func (zs *ZitadelService) GetAuthorizationURL(state, redirectURI string) string {
	if !zs.configured {
		return ""
	}

	// Build authorization URL
	return fmt.Sprintf(
		"%s/oauth/v2/authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s&scope=openid+profile+email",
		strings.TrimSuffix(zs.issuer, "/"),
		zs.clientID,
		redirectURI,
		state,
	)
}

// GetDiscoveryEndpoint returns the OIDC discovery endpoint
func (zs *ZitadelService) GetDiscoveryEndpoint() string {
	if !zs.configured {
		return ""
	}
	return strings.TrimSuffix(zs.issuer, "/") + "/.well-known/openid-configuration"
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func (zs *ZitadelService) ExchangeCodeForToken(
	ctx context.Context,
	req TokenExchangeRequest,
) (*TokenExchangeResponse, error) {
	if !zs.configured {
		return nil, fmt.Errorf("zitadel service not configured")
	}

	log := zs.log.Function("ExchangeCodeForToken")

	// Create token exchange request
	tokenURL := strings.TrimSuffix(zs.issuer, "/") + "/oauth/v2/token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", req.Code)
	data.Set("redirect_uri", req.RedirectURI)
	data.Set("client_id", zs.clientID)

	// Only add client secret if it's configured (for confidential clients)
	if zs.clientSecret != "" {
		data.Set("client_secret", zs.clientSecret)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		tokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, log.Err("failed to create token exchange request", err)
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := zs.httpClient.Do(httpReq)
	if err != nil {
		return nil, log.Err("failed to make token exchange request", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Info("failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, log.Err(
			"token exchange request failed",
			fmt.Errorf("status code: %d", resp.StatusCode),
		)
	}

	var tokenResp TokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, log.Err("failed to decode token response", err)
	}

	log.Info("token exchange successful", "tokenType", tokenResp.TokenType)
	return &tokenResp, nil
}

// generateJWTAssertion creates a JWT assertion for machine-to-machine authentication
func (zs *ZitadelService) generateJWTAssertion() (string, error) {
	if zs.privateKey == nil {
		return "", fmt.Errorf("private key not configured")
	}

	log := zs.log.Function("generateJWTAssertion")

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    zs.clientIDM2M, // Use machine-to-machine clientId from zitadel.json
		Subject:   zs.clientIDM2M, // Use machine-to-machine clientId from zitadel.json
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

