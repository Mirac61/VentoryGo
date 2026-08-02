package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterNormalizesEmail(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"  Max@Example.COM  ", "max@example.com"},
		{"max@example.com", "max@example.com"},
		{"\tMAX@EXAMPLE.COM\n", "max@example.com"},
	} {
		repo := &fakeRepo{}
		user, err := NewService(repo).Register(context.Background(), tc.in, "correct horse battery")
		require.NoError(t, err)

		assert.Equal(t, tc.want, user.Email, "Rueckgabe fuer %q", tc.in)
		require.Len(t, repo.created, 1)
		assert.Equal(t, tc.want, repo.created[0].Email, "gespeichert fuer %q", tc.in)
	}
}
