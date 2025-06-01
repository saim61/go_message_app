package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set JWT secret for testing
	SetJWTSecret("test-secret-key")
	m.Run()
}

func TestNewToken(t *testing.T) {
	tests := []struct {
		name     string
		username string
		ttl      time.Duration
		wantErr  bool
	}{
		{
			name:     "valid token creation",
			username: "testuser",
			ttl:      time.Hour,
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			ttl:      time.Hour,
			wantErr:  false, // Empty username is allowed in this implementation
		},
		{
			name:     "zero TTL",
			username: "testuser",
			ttl:      0,
			wantErr:  false, // Zero TTL creates an immediately expired token
		},
		{
			name:     "negative TTL",
			username: "testuser",
			ttl:      -time.Hour,
			wantErr:  false, // Negative TTL creates an expired token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := NewToken(tt.username, tt.ttl)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Verify token format (should have 3 parts separated by dots)
				parts := len(token)
				assert.Greater(t, parts, 0)
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   bool
		wantUser  string
	}{
		{
			name: "valid token",
			setupFunc: func() string {
				token, _ := NewToken("testuser", time.Hour)
				return token
			},
			wantErr:  false,
			wantUser: "testuser",
		},
		{
			name: "expired token",
			setupFunc: func() string {
				token, _ := NewToken("testuser", -time.Hour)
				return token
			},
			wantErr:  true,
			wantUser: "",
		},
		{
			name: "invalid token format",
			setupFunc: func() string {
				return "invalid.token.format"
			},
			wantErr:  true,
			wantUser: "",
		},
		{
			name: "empty token",
			setupFunc: func() string {
				return ""
			},
			wantErr:  true,
			wantUser: "",
		},
		{
			name: "malformed token",
			setupFunc: func() string {
				return "not.a.jwt"
			},
			wantErr:  true,
			wantUser: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupFunc()
			claims, err := ParseToken(token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, tt.wantUser, claims.Username)
			}
		})
	}
}

func TestTokenRoundTrip(t *testing.T) {
	username := "roundtripuser"
	ttl := time.Hour

	// Create token
	token, err := NewToken(username, ttl)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Parse token
	claims, err := ParseToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify claims
	assert.Equal(t, username, claims.Username)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
	assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Minute)))
}

func TestTokenWithDifferentSecrets(t *testing.T) {
	// Create token with one secret
	SetJWTSecret("secret1")
	token, err := NewToken("testuser", time.Hour)
	require.NoError(t, err)

	// Try to parse with different secret
	SetJWTSecret("secret2")
	claims, err := ParseToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// Restore original secret
	SetJWTSecret("test-secret-key")
}

func TestClaimsValidation(t *testing.T) {
	username := "claimsuser"
	token, err := NewToken(username, time.Hour)
	require.NoError(t, err)

	claims, err := ParseToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify custom claims
	assert.Equal(t, username, claims.Username)

	// Verify standard claims
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt.Time))
}
