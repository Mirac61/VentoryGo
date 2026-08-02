package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordFormat(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	require.NoError(t, err)

	// Ohne testify waere das: if !strings.HasPrefix(...) { t.Errorf(...) }
	assert.True(t, strings.HasPrefix(hash, "$argon2id$"), "hash %q ist kein PHC-String", hash)
}

func TestVerifyPasswordAcceptsCorrectPassword(t *testing.T) {
	const pw = "correct horse battery"

	hash, err := HashPassword(pw)
	require.NoError(t, err)

	assert.NoError(t, VerifyPassword(hash, pw))
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	require.NoError(t, err)

	assert.ErrorIs(t, VerifyPassword(hash, "falsch"), ErrMismatchedPassword)
}

// Gleiches Passwort, zwei Aufrufe, zwei verschiedene Hashes: der Salt wirkt.
func TestHashPasswordUsesFreshSalt(t *testing.T) {
	const pw = "correct horse battery"

	first, err := HashPassword(pw)
	require.NoError(t, err)
	second, err := HashPassword(pw)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.NoError(t, VerifyPassword(second, pw))
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, hash := range []string{
		"",
		"kein-phc-string",
		"$bcrypt$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",
	} {
		assert.ErrorIs(t, VerifyPassword(hash, "egal"), ErrInvalidHash, "hash=%q", hash)
	}
}
