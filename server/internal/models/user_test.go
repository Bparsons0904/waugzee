package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUser_IsOIDCUser(t *testing.T) {
	tests := []struct {
		name       string
		oidcUserID string
		expected   bool
	}{
		{
			name:       "User with OIDC ID",
			oidcUserID: "123456",
			expected:   true,
		},
		{
			name:       "User without OIDC ID",
			oidcUserID: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				OIDCUserID: tt.oidcUserID,
			}
			assert.Equal(t, tt.expected, user.IsOIDCUser())
		})
	}
}

func TestUser_UpdateFromOIDC(t *testing.T) {
	t.Run("Updates all fields correctly", func(t *testing.T) {
		user := &User{}
		email := "test@example.com"
		name := "Test User"
		provider := "zitadel"
		projectID := "project123"

		user.UpdateFromOIDC(
			"oidc123",
			&email,
			&name,
			"Test",
			"User",
			provider,
			&projectID,
			true,
		)

		assert.Equal(t, "oidc123", user.OIDCUserID)
		assert.Equal(t, &email, user.Email)
		assert.Equal(t, "Test", user.FirstName)
		assert.Equal(t, "User", user.LastName)
		assert.Equal(t, "Test User", user.FullName)
		assert.Equal(t, name, user.DisplayName)
		assert.Equal(t, &provider, user.OIDCProvider)
		assert.Equal(t, &projectID, user.OIDCProjectID)
		assert.True(t, user.ProfileVerified)
		assert.NotNil(t, user.LastLoginAt)
		assert.WithinDuration(t, time.Now(), *user.LastLoginAt, time.Second)
	})

	t.Run("Handles nil email", func(t *testing.T) {
		user := &User{}
		user.UpdateFromOIDC("oidc123", nil, nil, "Test", "User", "zitadel", nil, false)

		assert.Nil(t, user.Email)
		assert.Equal(t, "Test", user.FirstName)
		assert.Equal(t, "User", user.LastName)
	})

	t.Run("Uses FullName for DisplayName when name is nil", func(t *testing.T) {
		user := &User{}
		user.UpdateFromOIDC("oidc123", nil, nil, "Test", "User", "zitadel", nil, false)

		assert.Equal(t, "Test User", user.FullName)
		assert.Equal(t, "Test User", user.DisplayName)
	})

	t.Run("Profile verified only when email verified and present", func(t *testing.T) {
		user := &User{}
		email := "test@example.com"

		user.UpdateFromOIDC("oidc123", &email, nil, "Test", "User", "zitadel", nil, true)
		assert.True(t, user.ProfileVerified)

		user2 := &User{}
		user2.UpdateFromOIDC("oidc123", nil, nil, "Test", "User", "zitadel", nil, true)
		assert.False(t, user2.ProfileVerified)
	})
}
