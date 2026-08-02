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
