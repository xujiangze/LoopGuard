package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", h)
	assert.True(t, VerifyPassword(h, "secret123"))
	assert.False(t, VerifyPassword(h, "wrong"))
}

func TestJWTSignParse(t *testing.T) {
	secret := "test-secret"
	tok, err := SignJWT(secret, 42, "admin")
	require.NoError(t, err)
	claims, err := ParseJWT(secret, tok)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), claims.UserID)
	assert.Equal(t, "admin", claims.Role)

	_, err = ParseJWT("wrong-secret", tok)
	assert.Error(t, err)
}

func TestAPIKeyGenerateHashVerify(t *testing.T) {
	plain := GenerateAPIKey()
	assert.True(t, len(plain) >= 32)
	h := HashAPIKey(plain)
	assert.Equal(t, h, HashAPIKey(plain))
	assert.NotEqual(t, plain, h)
}
