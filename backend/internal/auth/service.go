package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const sessionTTL = 30 * 24 * time.Hour

var dummyHash = sync.OnceValue(func() string {
	hash, _ := HashPassword("dummy")
	return hash
})

type userRepository interface {
	Create(ctx context.Context, user User) error
	FindByEmail(ctx context.Context, email string) (User, error)
}

type Service struct {
	repo     userRepository
	sessions SessionStore
}

func NewService(repo userRepository, sessions SessionStore) *Service {
	return &Service{repo: repo, sessions: sessions}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)

	hash, err := HashPassword(password)
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
			_ = VerifyPassword(dummyHash(), password)
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("find user: %w", err)
	}

	if err := VerifyPassword(user.PasswordHash, password); err != nil {
		if errors.Is(err, ErrMismatchedPassword) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("verify password: %w", err)
	}
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return User{}, "", fmt.Errorf("parse user id: %w", err)
	}
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return User{}, "", fmt.Errorf("new session token: %w", err)
	}
	now := time.Now().UTC()
	session := Session{
		TokenHash: tokenHash,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionTTL),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return User{}, "", fmt.Errorf("create session: %w", err)
	}

	return user, token, nil
}
