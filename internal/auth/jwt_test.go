package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTokenType(t *testing.T) {
	svc := NewJWTService("test-secret", 2, 7)

	accessToken, err := svc.GenerateToken("u1", "admin", "super_admin")
	require.NoError(t, err)

	refreshToken, err := svc.GenerateRefreshToken("u1", "admin", "super_admin")
	require.NoError(t, err)

	accessClaims, err := svc.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, "access", accessClaims.TokenType)

	_, err = svc.ValidateToken(refreshToken)
	assert.ErrorIs(t, err, ErrInvalidTokenType)

	refreshClaims, err := svc.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, "refresh", refreshClaims.TokenType)

	_, err = svc.ValidateRefreshToken(accessToken)
	assert.ErrorIs(t, err, ErrInvalidTokenType)
}
