package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, user User) error {
	const query = `INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, user.ID, user.Email, user.PasswordHash, user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	const query = `SELECT id,email, password_hash, created_at FROM users WHERE lower(email)=$1`
	err := r.pool.QueryRow(ctx, query, strings.ToLower(email)).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

type PostgresSessionStore struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionStore(pool *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{pool: pool}
}

func (r *PostgresSessionStore) Create(ctx context.Context, s Session) error {
	const query = `INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, hex.EncodeToString(s.TokenHash), s.UserID, s.CreatedAt, s.ExpiresAt)
	return err
}

func (r *PostgresSessionStore) Get(ctx context.Context, tokenHash []byte) (Session, error) {
	const query = `
		WITH expired AS (
			DELETE FROM sessions WHERE token_hash = $1 AND expires_at <= now()
		)
		SELECT user_id, created_at, expires_at FROM sessions
		WHERE token_hash = $1 AND expires_at > now()`
	s := Session{TokenHash: tokenHash}
	err := r.pool.QueryRow(ctx, query, hex.EncodeToString(tokenHash)).Scan(&s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, err
	}
	return s, nil
}

func (r *PostgresSessionStore) Touch(ctx context.Context, tokenHash []byte, expiresAt time.Time) error {
	const query = `UPDATE sessions SET expires_at = $2 WHERE token_hash = $1`
	_, err := r.pool.Exec(ctx, query, hex.EncodeToString(tokenHash), expiresAt)
	return err
}

func (r *PostgresSessionStore) Delete(ctx context.Context, tokenHash []byte) error {
	const query = `Delete FROM sessions WHERE token_hash=$1`
	_, err := r.pool.Exec(ctx, query, hex.EncodeToString(tokenHash))
	return err
}

func (r *PostgresSessionStore) DeleteExpiredByUser(ctx context.Context, userID uuid.UUID) error {
	const query = `DELETE FROM sessions WHERE user_id = $1 AND expires_at <= now()`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}
