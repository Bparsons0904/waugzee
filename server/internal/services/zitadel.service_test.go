package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"waugzee/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZitadelService_RevokeToken(t *testing.T) {
	// Mock OIDC discovery server
	var serverURL string
	discoveryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/.well-known/openid-configuration") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Use the server URL in the response to match what the service expects
			response := `{
				"issuer": "` + serverURL + `",
				"authorization_endpoint": "` + serverURL + `/oauth/v2/authorize",
				"token_endpoint": "` + serverURL + `/oauth/v2/token",
				"userinfo_endpoint": "` + serverURL + `/oidc/v1/userinfo",
				"jwks_uri": "` + serverURL + `/oauth/v2/keys",
				"introspection_endpoint": "` + serverURL + `/oauth/v2/introspect",
				"revocation_endpoint": "` + serverURL + `/oauth/v2/revoke",
				"end_session_endpoint": "` + serverURL + `/oidc/v1/end_session"
			}`
			_, _ = w.Write([]byte(response))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/oauth/v2/revoke") {
			// Validate form data
			err := r.ParseForm()
			require.NoError(t, err)

			token := r.FormValue("token")
			assert.Equal(t, "test-access-token", token)

			tokenTypeHint := r.FormValue("token_type_hint")
			assert.Equal(t, "access_token", tokenTypeHint)

			clientID := r.FormValue("client_id")
			assert.Equal(t, "test-client-id", clientID)

			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(discoveryHandler)
	defer server.Close()

	// Set the server URL for the discovery response
	serverURL = server.URL

	// Create service with mock server
	cfg := config.Config{
		ZitadelInstanceURL:  server.URL,
		ZitadelClientID:     "test-client-id",
		ZitadelClientSecret: "test-client-secret",
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.True(t, service.IsConfigured())

	ctx := context.Background()

	// Test token revocation
	err = service.RevokeToken(ctx, "test-access-token", "access_token")
	assert.NoError(t, err)
}

func TestZitadelService_GetLogoutURL(t *testing.T) {
	// Mock OIDC discovery server
	var serverURL string
	discoveryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/.well-known/openid-configuration") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Use the server URL in the response to match what the service expects
			response := `{
				"issuer": "` + serverURL + `",
				"authorization_endpoint": "` + serverURL + `/oauth/v2/authorize",
				"token_endpoint": "` + serverURL + `/oauth/v2/token",
				"userinfo_endpoint": "` + serverURL + `/oidc/v1/userinfo",
				"jwks_uri": "` + serverURL + `/oauth/v2/keys",
				"introspection_endpoint": "` + serverURL + `/oauth/v2/introspect",
				"revocation_endpoint": "` + serverURL + `/oauth/v2/revoke",
				"end_session_endpoint": "` + serverURL + `/oidc/v1/end_session"
			}`
			_, _ = w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(discoveryHandler)
	defer server.Close()

	// Set the server URL for the discovery response
	serverURL = server.URL

	// Create service with mock server
	cfg := config.Config{
		ZitadelInstanceURL:  server.URL,
		ZitadelClientID:     "test-client-id",
		ZitadelClientSecret: "test-client-secret",
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.True(t, service.IsConfigured())

	ctx := context.Background()

	tests := []struct {
		name                  string
		idTokenHint           string
		postLogoutRedirectURI string
		state                 string
		expectedContains      []string
	}{
		{
			name:                  "basic logout URL",
			idTokenHint:           "",
			postLogoutRedirectURI: "",
			state:                 "",
			expectedContains:      []string{server.URL + "/oidc/v1/end_session"},
		},
		{
			name:                  "logout URL with all parameters",
			idTokenHint:           "test-id-token",
			postLogoutRedirectURI: "https://app.example.com/logout",
			state:                 "test-state",
			expectedContains: []string{
				server.URL + "/oidc/v1/end_session",
				"id_token_hint=test-id-token",
				"post_logout_redirect_uri=https%3A%2F%2Fapp.example.com%2Flogout",
				"state=test-state",
			},
		},
		{
			name:                  "logout URL with ID token hint only",
			idTokenHint:           "test-id-token",
			postLogoutRedirectURI: "",
			state:                 "",
			expectedContains: []string{
				server.URL + "/oidc/v1/end_session",
				"id_token_hint=test-id-token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logoutURL, err := service.GetLogoutURL(
				ctx,
				tt.idTokenHint,
				tt.postLogoutRedirectURI,
				tt.state,
			)
			require.NoError(t, err)

			for _, expectedContent := range tt.expectedContains {
				assert.Contains(
					t,
					logoutURL,
					expectedContent,
					"Expected logout URL to contain: %s",
					expectedContent,
				)
			}
		})
	}
}

func TestZitadelService_RevokeToken_NotConfigured(t *testing.T) {
	// Test with unconfigured service
	cfg := config.Config{
		ZitadelInstanceURL: "",
		ZitadelClientID:    "",
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.False(t, service.IsConfigured())

	ctx := context.Background()

	err = service.RevokeToken(ctx, "token", "access_token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "zitadel service not configured")
}

func TestZitadelService_GetLogoutURL_NotConfigured(t *testing.T) {
	// Test with unconfigured service
	cfg := config.Config{
		ZitadelInstanceURL: "",
		ZitadelClientID:    "",
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.False(t, service.IsConfigured())

	ctx := context.Background()

	logoutURL, err := service.GetLogoutURL(ctx, "", "", "")
	assert.Error(t, err)
	assert.Empty(t, logoutURL)
	assert.Contains(t, err.Error(), "zitadel service not configured")
}

func TestZitadelService_GetAuthorizationURL_PKCE(t *testing.T) {
	// Create service for testing authorization URL generation
	cfg := config.Config{
		ZitadelInstanceURL:  "https://test.zitadel.dev",
		ZitadelClientID:     "test-client-id",
		ZitadelClientSecret: "", // No client secret for public client
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.True(t, service.IsConfigured())

	tests := []struct {
		name             string
		state            string
		redirectURI      string
		codeChallenge    string
		expectedContains []string
		notContains      []string
	}{
		{
			name:          "authorization URL without PKCE",
			state:         "test-state",
			redirectURI:   "https://app.example.com/callback",
			codeChallenge: "",
			expectedContains: []string{
				"https://test.zitadel.dev/oauth/v2/authorize",
				"client_id=test-client-id",
				"response_type=code",
				"redirect_uri=https://app.example.com/callback",
				"state=test-state",
				"scope=openid+profile+email",
			},
			notContains: []string{
				"code_challenge=",
				"code_challenge_method=",
			},
		},
		{
			name:          "authorization URL with PKCE",
			state:         "test-state",
			redirectURI:   "https://app.example.com/callback",
			codeChallenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			expectedContains: []string{
				"https://test.zitadel.dev/oauth/v2/authorize",
				"client_id=test-client-id",
				"response_type=code",
				"redirect_uri=https://app.example.com/callback",
				"state=test-state",
				"scope=openid+profile+email",
				"code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
				"code_challenge_method=S256",
			},
			notContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL := service.GetAuthorizationURL(tt.state, tt.redirectURI, tt.codeChallenge, "")
			require.NotEmpty(t, authURL)

			for _, expectedContent := range tt.expectedContains {
				assert.Contains(
					t,
					authURL,
					expectedContent,
					"Expected authorization URL to contain: %s",
					expectedContent,
				)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(
					t,
					authURL,
					notContains,
					"Expected authorization URL to NOT contain: %s",
					notContains,
				)
			}
		})
	}
}

func TestZitadelService_ExchangeCodeForToken_PKCE(t *testing.T) {
	// Mock token exchange server
	tokenHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/v2/token") {
			// Validate form data
			err := r.ParseForm()
			require.NoError(t, err)

			grantType := r.FormValue("grant_type")
			assert.Equal(t, "authorization_code", grantType)

			code := r.FormValue("code")
			assert.Equal(t, "test-auth-code", code)

			redirectURI := r.FormValue("redirect_uri")
			assert.Equal(t, "https://app.example.com/callback", redirectURI)

			clientID := r.FormValue("client_id")
			assert.Equal(t, "test-client-id", clientID)

			// Check for PKCE parameters
			codeVerifier := r.FormValue("code_verifier")
			clientSecret := r.FormValue("client_secret")

			// Should have code verifier but no client secret for PKCE flow
			assert.Equal(t, "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", codeVerifier)
			assert.Empty(t, clientSecret, "Client secret should not be present in PKCE flow")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"access_token": "test-access-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"id_token": "test-id-token"
			}`
			_, _ = w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(tokenHandler)
	defer server.Close()

	// Create service with mock server
	cfg := config.Config{
		ZitadelInstanceURL:  server.URL,
		ZitadelClientID:     "test-client-id",
		ZitadelClientSecret: "", // No client secret for public client
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.True(t, service.IsConfigured())

	ctx := context.Background()

	// Test PKCE token exchange
	req := TokenExchangeRequest{
		Code:         "test-auth-code",
		RedirectURI:  "https://app.example.com/callback",
		State:        "test-state",
		CodeVerifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", // PKCE code verifier
	}

	tokenResp, err := service.ExchangeCodeForToken(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, tokenResp)

	assert.Equal(t, "test-access-token", tokenResp.AccessToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Equal(t, int64(3600), tokenResp.ExpiresIn)
	assert.Equal(t, "test-id-token", tokenResp.IDToken)
}

func TestZitadelService_ExchangeCodeForToken_ConfidentialClient(t *testing.T) {
	// Mock token exchange server for confidential client
	tokenHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/v2/token") {
			// Validate form data
			err := r.ParseForm()
			require.NoError(t, err)

			grantType := r.FormValue("grant_type")
			assert.Equal(t, "authorization_code", grantType)

			code := r.FormValue("code")
			assert.Equal(t, "test-auth-code", code)

			redirectURI := r.FormValue("redirect_uri")
			assert.Equal(t, "https://app.example.com/callback", redirectURI)

			clientID := r.FormValue("client_id")
			assert.Equal(t, "test-client-id", clientID)

			// Check for client secret (no PKCE for confidential clients)
			codeVerifier := r.FormValue("code_verifier")
			clientSecret := r.FormValue("client_secret")

			// Should have client secret but no code verifier for confidential client
			assert.Empty(
				t,
				codeVerifier,
				"Code verifier should not be present for confidential client",
			)
			assert.Equal(t, "test-client-secret", clientSecret)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := `{
				"access_token": "test-access-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"id_token": "test-id-token"
			}`
			_, _ = w.Write([]byte(response))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(tokenHandler)
	defer server.Close()

	// Create service with mock server (confidential client)
	cfg := config.Config{
		ZitadelInstanceURL:  server.URL,
		ZitadelClientID:     "test-client-id",
		ZitadelClientSecret: "test-client-secret", // Has client secret
	}

	service, err := NewZitadelService(cfg)
	require.NoError(t, err)
	require.True(t, service.IsConfigured())

	ctx := context.Background()

	// Test confidential client token exchange (no PKCE)
	req := TokenExchangeRequest{
		Code:         "test-auth-code",
		RedirectURI:  "https://app.example.com/callback",
		State:        "test-state",
		CodeVerifier: "", // No code verifier for confidential client
	}

	tokenResp, err := service.ExchangeCodeForToken(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, tokenResp)

	assert.Equal(t, "test-access-token", tokenResp.AccessToken)
	assert.Equal(t, "Bearer", tokenResp.TokenType)
	assert.Equal(t, int64(3600), tokenResp.ExpiresIn)
	assert.Equal(t, "test-id-token", tokenResp.IDToken)
}

