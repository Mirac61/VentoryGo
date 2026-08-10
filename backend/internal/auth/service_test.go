package auth

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceRejectsMissingSessionStore(t *testing.T) {
	assert.Panics(t, func() { NewService(&fakeRepo{}, nil) })
}

func TestLoginStoresOnlyTheTokenHash(t *testing.T) {
	store := &fakeSessionStore{}
	service := NewService(&fakeRepo{}, store)

	registered, err := service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	user, token, err := service.Login(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	assert.Equal(t, registered.ID, user.ID)
	require.NotEmpty(t, token)
	require.Len(t, store.created, 1)

	session := store.created[0]
	want := sha256.Sum256([]byte(token))
	assert.Equal(t, want[:], session.TokenHash, "in der DB darf nur der Hash liegen")
	assert.NotContains(t, string(session.TokenHash), token)
	assert.Equal(t, registered.ID, session.UserID.String())
	assert.WithinDuration(t, session.CreatedAt.Add(defaultSessionTTL), session.ExpiresAt, time.Second)
}

func TestSessionTTLFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "nicht gesetzt", want: defaultSessionTTL},
		{name: "kurz fuer Tests", value: "1s", want: time.Second},
		{name: "sieben Tage", value: "168h", want: 168 * time.Hour},
		{name: "kaputt", value: "30 Tage", wantErr: true},
		{name: "null", value: "0s", wantErr: true},
		{name: "negativ", value: "-1h", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SESSION_TTL", tc.value)

			ttl, err := SessionTTLFromEnv()

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, ttl)
		})
	}
}

func TestLoginUsesConfiguredSessionTTL(t *testing.T) {
	store := &fakeSessionStore{}
	hasher, err := NewHasher(1)
	require.NoError(t, err)
	service := NewServiceWithSessionTTL(&fakeRepo{}, store, time.Second, hasher)

	_, err = service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)
	_, _, err = service.Login(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	require.Len(t, store.created, 1)
	session := store.created[0]
	assert.WithinDuration(t, session.CreatedAt.Add(time.Second), session.ExpiresAt, time.Millisecond*100)
}

func TestLoginClearsExpiredSessionsOfTheSameUser(t *testing.T) {
	service, store := serviceWithUser(t, "max@example.com", "correct horse battery")

	user, _, err := service.Login(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	assert.Equal(t, []uuid.UUID{uuid.MustParse(user.ID)}, store.expiredCleared)
}

func TestLoginHidesWhetherAccountExists(t *testing.T) {
	service := NewService(&fakeRepo{}, &fakeSessionStore{})
	_, err := service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	for _, tc := range []struct{ name, email, password string }{
		{"falsches Passwort", "max@example.com", "falsch aber lang"},
		{"unbekannte Mail", "nobody@example.com", "correct horse battery"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, token, err := service.Login(context.Background(), tc.email, tc.password)

			require.ErrorIs(t, err, ErrInvalidCredentials)
			assert.NotErrorIs(t, err, ErrUserNotFound, "ErrUserNotFound darf den Service nicht verlassen")
			assert.Empty(t, token)
		})
	}
}

func TestLoginPropagatesContextError(t *testing.T) {
	service, _ := serviceWithUser(t, "max@example.com", "correct horse battery")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Beide Zweige muessen sich gleich verhalten, sonst verraet die Antwort,
	// ob es das Konto gibt.
	for _, email := range []string{"max@example.com", "nobody@example.com"} {
		_, token, err := service.Login(ctx, email, "correct horse battery")

		assert.ErrorIs(t, err, context.Canceled, "Login fuer %q", email)
		assert.NotErrorIs(t, err, ErrInvalidCredentials, "Login fuer %q", email)
		assert.Empty(t, token, "Login fuer %q", email)
	}
}

func TestLoginNormalizesEmail(t *testing.T) {
	service := NewService(&fakeRepo{}, &fakeSessionStore{})
	_, err := service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	for _, in := range []string{"  Max@Example.COM  ", "MAX@EXAMPLE.COM", "\tmax@example.com\n"} {
		_, token, err := service.Login(context.Background(), in, "correct horse battery")
		require.NoError(t, err, "Login fuer %q", in)
		assert.NotEmpty(t, token)
	}
}

func TestLoginIssuesFreshTokenPerCall(t *testing.T) {
	store := &fakeSessionStore{}
	service := NewService(&fakeRepo{}, store)
	_, err := service.Register(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	_, first, err := service.Login(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)
	_, second, err := service.Login(context.Background(), "max@example.com", "correct horse battery")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.Len(t, store.created, 2, "paralleles Anmelden darf bestehende Sessions nicht ersetzen")
}

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
		user, err := NewService(repo, &fakeSessionStore{}).Register(context.Background(), tc.in, "correct horse battery")
		require.NoError(t, err)

		assert.Equal(t, tc.want, user.Email, "Rueckgabe fuer %q", tc.in)
		require.Len(t, repo.created, 1)
		assert.Equal(t, tc.want, repo.created[0].Email, "gespeichert fuer %q", tc.in)
	}
}
