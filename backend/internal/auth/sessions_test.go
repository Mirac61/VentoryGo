package auth

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionTokenIsUrlSafeAndDecodesTo32Bytes(t *testing.T) {
	token, hash, err := newSessionToken()
	require.NoError(t, err)

	raw, err := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, err, "Token %q ist kein base64url ohne Padding", token)

	assert.Len(t, raw, 32)
	assert.Len(t, hash, 32)
	assert.NotContains(t, token, "=")
	assert.NotContains(t, token, "+")
	assert.NotContains(t, token, "/")
}

func TestNewSessionTokenHashMatchesHashToken(t *testing.T) {
	token, hash, err := newSessionToken()
	require.NoError(t, err)

	assert.Equal(t, hash, hashToken(token))
}

func TestNewSessionTokenReturnsDistinctTokens(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		token, _, err := newSessionToken()
		require.NoError(t, err)

		_, duplicate := seen[token]
		require.False(t, duplicate, "Token %q wurde zweimal erzeugt", token)
		seen[token] = struct{}{}
	}
}

func TestHashTokenRejectsManipulatedToken(t *testing.T) {
	token, hash, err := newSessionToken()
	require.NoError(t, err)

	assert.NotEqual(t, hash, hashToken(token+"x"))
}
