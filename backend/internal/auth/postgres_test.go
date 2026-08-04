package auth

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Mirac61/VentoryGo/backend/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDSN(t *testing.T) string {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}

	for _, key := range []string{"TEST_DATABASE_URL", "DATABASE_URL"} {
		if dsn := os.Getenv(key); dsn != "" {
			return dsn
		}
	}
	t.Fatal("TEST_DATABASE_URL or DATABASE_URL must be set, or run with -short")
	return ""
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), testDSN(t))
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func testEmail(local string) string {
	return local + "+" + uuid.NewString() + "@example.test"
}

func newTestUser(t *testing.T, email string) User {
	t.Helper()

	hash, err := HashPassword("correct horse battery")
	require.NoError(t, err)

	return User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
}

func createTestUser(t *testing.T, repo *PostgresRepository, pool *pgxpool.Pool, user User) error {
	t.Helper()

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
	})
	return repo.Create(context.Background(), user)
}

func TestPostgresCreateStoresUser(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)

	user := newTestUser(t, testEmail("stores"))
	require.NoError(t, createTestUser(t, repo, pool, user))

	var email, storedHash string
	err := pool.QueryRow(context.Background(),
		`SELECT email, password_hash FROM users WHERE id = $1`, user.ID).Scan(&email, &storedHash)
	require.NoError(t, err)

	assert.Equal(t, user.Email, email)
	assert.True(t, strings.HasPrefix(storedHash, "$argon2id$"),
		"gespeicherter Hash %q ist kein Argon2id-PHC-String", storedHash)
}

func TestPostgresCreateRejectsDuplicateEmailIgnoringCase(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)

	email := testEmail("duplicate")

	first := newTestUser(t, strings.ToUpper(email))
	require.NoError(t, createTestUser(t, repo, pool, first))

	second := newTestUser(t, email)
	require.NotEqual(t, first.ID, second.ID)

	assert.ErrorIs(t, createTestUser(t, repo, pool, second), ErrEmailTaken)
}

func TestPostgresFindByEmailIgnoresCase(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresRepository(pool)

	email := testEmail("findme")
	user := newTestUser(t, strings.ToUpper(email))
	require.NoError(t, createTestUser(t, repo, pool, user))

	found, err := repo.FindByEmail(context.Background(), email)
	require.NoError(t, err)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.PasswordHash, found.PasswordHash)
	assert.WithinDuration(t, user.CreatedAt, found.CreatedAt, time.Second)
}

func TestPostgresFindByEmailReportsUnknownEmail(t *testing.T) {
	repo := NewPostgresRepository(testPool(t))

	_, err := repo.FindByEmail(context.Background(), testEmail("nobody"))
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// Sessions haengen per ON DELETE CASCADE am User, das Cleanup aus createTestUser
// raeumt sie also mit ab.
func createSessionUser(t *testing.T, pool *pgxpool.Pool, local string) uuid.UUID {
	t.Helper()

	user := newTestUser(t, testEmail(local))
	require.NoError(t, createTestUser(t, NewPostgresRepository(pool), pool, user))

	id, err := uuid.Parse(user.ID)
	require.NoError(t, err)
	return id
}

func createTestSession(t *testing.T, store *PostgresSessionStore, userID uuid.UUID, expiresAt time.Time) []byte {
	t.Helper()

	_, hash, err := newSessionToken()
	require.NoError(t, err)

	require.NoError(t, store.Create(context.Background(), Session{
		TokenHash: hash,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	}))
	return hash
}

func countSessions(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()

	var count int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM sessions WHERE user_id = $1`, userID).Scan(&count))
	return count
}

func TestSessionStoreCreateAndGetRoundtrip(t *testing.T) {
	pool := testPool(t)
	store := NewPostgresSessionStore(pool)

	userID := createSessionUser(t, pool, "roundtrip")
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	hash := createTestSession(t, store, userID, expiresAt)

	got, err := store.Get(context.Background(), hash)
	require.NoError(t, err)

	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, hash, got.TokenHash)
	assert.WithinDuration(t, expiresAt, got.ExpiresAt, time.Second)
}

func TestSessionStoreGetRejectsUnknownToken(t *testing.T) {
	pool := testPool(t)
	store := NewPostgresSessionStore(pool)

	_, hash, err := newSessionToken()
	require.NoError(t, err)

	_, err = store.Get(context.Background(), hash)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionStoreGetRejectsExpiredSession(t *testing.T) {
	pool := testPool(t)
	store := NewPostgresSessionStore(pool)

	userID := createSessionUser(t, pool, "expired")
	hash := createTestSession(t, store, userID, time.Now().UTC().Add(-time.Minute))

	_, err := store.Get(context.Background(), hash)
	assert.ErrorIs(t, err, ErrSessionNotFound)

	assert.Zero(t, countSessions(t, pool, userID),
		"abgelaufene Session muss beim Zugriff geloescht werden")
}

func TestSessionStoreTouchExtendsExpiry(t *testing.T) {
	pool := testPool(t)
	store := NewPostgresSessionStore(pool)

	userID := createSessionUser(t, pool, "touch")
	hash := createTestSession(t, store, userID, time.Now().UTC().Add(time.Hour))

	extended := time.Now().UTC().Add(30 * 24 * time.Hour)
	require.NoError(t, store.Touch(context.Background(), hash, extended))

	got, err := store.Get(context.Background(), hash)
	require.NoError(t, err)
	assert.WithinDuration(t, extended, got.ExpiresAt, time.Second)
}

func TestSessionStoreDeleteEndsOnlyOneSession(t *testing.T) {
	pool := testPool(t)
	store := NewPostgresSessionStore(pool)

	userID := createSessionUser(t, pool, "twologins")
	expiresAt := time.Now().UTC().Add(time.Hour)
	laptop := createTestSession(t, store, userID, expiresAt)
	phone := createTestSession(t, store, userID, expiresAt)

	require.NoError(t, store.Delete(context.Background(), laptop))

	_, err := store.Get(context.Background(), laptop)
	assert.ErrorIs(t, err, ErrSessionNotFound)

	_, err = store.Get(context.Background(), phone)
	assert.NoError(t, err, "die zweite Session desselben Users muss bestehen bleiben")
}

func TestSessionStoreDeleteExpiredByUserSparesValidAndForeignSessions(t *testing.T) {
	pool := testPool(t)
	store := NewPostgresSessionStore(pool)

	valid := time.Now().UTC().Add(time.Hour)
	expired := time.Now().UTC().Add(-time.Hour)

	mine := createSessionUser(t, pool, "mine")
	createTestSession(t, store, mine, expired)
	myValidHash := createTestSession(t, store, mine, valid)

	other := createSessionUser(t, pool, "other")
	createTestSession(t, store, other, expired)

	require.NoError(t, store.DeleteExpiredByUser(context.Background(), mine))

	assert.Equal(t, 1, countSessions(t, pool, mine), "nur die abgelaufene eigene Session faellt weg")

	_, err := store.Get(context.Background(), myValidHash)
	assert.NoError(t, err, "die gueltige eigene Session muss bestehen bleiben")

	assert.Equal(t, 1, countSessions(t, pool, other), "fremde Sessions duerfen nicht mitgeloescht werden")
}
