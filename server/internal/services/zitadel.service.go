package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"waugzee/config"
	"waugzee/internal/logger"
)

// ZitadelService handles OIDC authentication and user management
type ZitadelService struct {
	config       config.Config
	log          logger.Logger
	httpClient   *http.Client
	issuer       string
	clientID     string
	clientSecret string
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
		configured:   true,
	}

	log.Info("Zitadel service initialized successfully", "issuer", cfg.ZitadelInstanceURL)
	return service, nil
}

// ValidateToken validates an access token using OIDC introspection
func (zs *ZitadelService) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	if !zs.configured {
		return nil, fmt.Errorf("zitadel service not configured")
	}

	log := zs.log.Function("ValidateToken")

	// Create introspection request
	introspectURL := strings.TrimSuffix(zs.issuer, "/") + "/oauth/v2/introspect"
	
	req, err := http.NewRequestWithContext(ctx, "POST", introspectURL, strings.NewReader(fmt.Sprintf("token=%s", token)))
	if err != nil {
		return nil, log.Err("failed to create introspection request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(zs.clientID, zs.clientSecret)

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
		return nil, log.Err("introspection request failed", fmt.Errorf("status code: %d", resp.StatusCode))
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
		ProjectID: zs.config.ZitadelProjectID,
		Valid:     true,
	}, nil
}

// GetUserInfo retrieves detailed user information by user ID
func (zs *ZitadelService) GetUserInfo(ctx context.Context, userID string) (interface{}, error) {
	if !zs.configured {
		return nil, fmt.Errorf("zitadel service not configured")
	}

	log := zs.log.Function("GetUserInfo")
	
	// This is a placeholder implementation
	// In a real implementation, you would call the Zitadel Management API
	log.Info("user info requested", "userID", userID)
	
	return map[string]interface{}{
		"id": userID,
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
	return fmt.Sprintf("%s/oauth/v2/authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s&scope=openid+profile+email",
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

// Close cleans up the Zitadel service resources
func (zs *ZitadelService) Close() error {
	// No resources to clean up for HTTP client
	return nil
}