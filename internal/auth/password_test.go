package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{
			name:        "valid password",
			password:    "password123",
			expectError: false,
		},
		{
			name:        "empty password",
			password:    "",
			expectError: false, // bcrypt allows empty passwords
		},
		{
			name:        "long password (over 72 bytes)",
			password:    strings.Repeat("a", 100), // 100 characters
			expectError: true,                     // bcrypt has a 72-byte limit
		},
		{
			name:        "special characters",
			password:    "p@ssw0rd!#$%",
			expectError: false,
		},
		{
			name:        "unicode password",
			password:    "пароль123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				assert.NotEqual(t, tt.password, hash) // Hash should be different from plain text
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	// Create a test hash for "password123"
	testPassword := "password123"
	testHash, err := HashPassword(testPassword)
	require.NoError(t, err)

	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  bool
	}{
		{
			name:     "correct password",
			hash:     testHash,
			password: testPassword,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			hash:     testHash,
			password: "wrongpassword",
			wantErr:  true,
		},
		{
			name:     "empty password with empty hash",
			hash:     "",
			password: "",
			wantErr:  true, // Invalid hash format
		},
		{
			name:     "empty password with valid hash",
			hash:     testHash,
			password: "",
			wantErr:  true,
		},
		{
			name:     "valid password with empty hash",
			hash:     "",
			password: testPassword,
			wantErr:  true, // Invalid hash format
		},
		{
			name:     "invalid hash format",
			hash:     "invalid-hash",
			password: testPassword,
			wantErr:  true,
		},
		{
			name:     "case sensitive password",
			hash:     testHash,
			password: "PASSWORD123",
			wantErr:  true, // Passwords are case sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPassword(tt.hash, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordHashUniqueness(t *testing.T) {
	password := "samepassword"

	// Generate multiple hashes for the same password
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)
	hash3, err3 := HashPassword(password)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// Each hash should be unique due to salt
	assert.NotEqual(t, hash1, hash2)
	assert.NotEqual(t, hash2, hash3)
	assert.NotEqual(t, hash1, hash3)

	// But all should verify correctly
	assert.NoError(t, CheckPassword(hash1, password))
	assert.NoError(t, CheckPassword(hash2, password))
	assert.NoError(t, CheckPassword(hash3, password))
}

func TestPasswordRoundTrip(t *testing.T) {
	passwords := []string{
		"simple",
		"complex!@#",
		"password with space",
		"unicode: 🔐",
		// Removed the long password test case since bcrypt has a 72-byte limit
	}

	for _, password := range passwords {
		t.Run("password_"+password, func(t *testing.T) {
			// Hash the password
			hash, err := HashPassword(password)
			require.NoError(t, err)
			require.NotEmpty(t, hash)

			// Verify the password
			err = CheckPassword(hash, password)
			assert.NoError(t, err, "Password verification should succeed")

			// Verify wrong password fails
			err = CheckPassword(hash, password+"wrong")
			assert.Error(t, err, "Wrong password verification should fail")
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
