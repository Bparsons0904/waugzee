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
	require.NotNil(t, service)

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
	require.NotNil(t, service)

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

func TestZitadelService_NotConfigured(t *testing.T) {
	// Test with unconfigured service - should fail at creation
	cfg := config.Config{
		ZitadelInstanceURL: "",
		ZitadelClientID:    "",
	}

	service, err := NewZitadelService(cfg)
	require.Error(t, err)
	require.Nil(t, service)
	assert.Contains(t, err.Error(), "missing ZitadelInstanceURL or ZitadelClientID")
}


