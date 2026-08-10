package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultSessionTTL = 30 * 24 * time.Hour

func SessionTTLFromEnv() (time.Duration, error) {
	value := os.Getenv("SESSION_TTL")
	if value == "" {
		return defaultSessionTTL, nil
	}

	ttl, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid SESSION_TTL %q: %w", value, err)
	}
	if ttl <= 0 {
		return 0, fmt.Errorf("invalid SESSION_TTL %q: muss positiv sein", value)
	}
	return ttl, nil
}

var dummyHash = sync.OnceValue(func() string {
	hash, _ := hashPassword("dummy")
	return hash
})

type userRepository interface {
	Create(ctx context.Context, user User) error
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id uuid.UUID) (User, error)
}

type Service struct {
	repo       userRepository
	sessions   SessionStore
	sessionTTL time.Duration
	hasher     *Hasher
}

func NewService(repo userRepository, sessions SessionStore) *Service {
	hasher, err := NewHasher(defaultHashConcurrency())
	if err != nil {
		panic(fmt.Sprintf("auth: %v", err))
	}
	return NewServiceWithSessionTTL(repo, sessions, defaultSessionTTL, hasher)
}

func NewServiceWithSessionTTL(repo userRepository, sessions SessionStore, ttl time.Duration, hasher *Hasher) *Service {
	if sessions == nil {
		panic("auth: sessions store is nil")
	}
	if hasher == nil {
		panic("auth: hasher is nil")
	}
	return &Service{repo: repo, sessions: sessions, sessionTTL: ttl, hasher: hasher}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	hash, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	user := User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	user, err := s.repo.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			_ = s.hasher.Verify(ctx, dummyHash(), password)
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("find user: %w", err)
	}

	if err := s.hasher.Verify(ctx, user.PasswordHash, password); err != nil {
		if errors.Is(err, ErrMismatchedPassword) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("verify password: %w", err)
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return User{}, "", fmt.Errorf("parse user id: %w", err)
	}
	_ = s.sessions.DeleteExpiredByUser(ctx, userID)

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return User{}, "", fmt.Errorf("new session token: %w", err)
	}
	now := time.Now().UTC()
	session := Session{
		TokenHash: tokenHash,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(s.sessionTTL),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return User{}, "", fmt.Errorf("create session: %w", err)
	}

	return user, token, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}

func (s *Service) Me(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.FindByID(ctx, id)
}
