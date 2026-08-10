package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameter aus ADR 0001 (OWASP-Empfehlung).
const (
	argonMemory      uint32 = 19 * 1024 // KiB
	argonTime        uint32 = 2
	argonParallelism uint8  = 1
	argonSaltLen     uint32 = 16
	argonKeyLen      uint32 = 32
)

type Hasher struct {
	sem chan struct{}
}

func NewHasher(concurrency int) (*Hasher, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("auth: concurrency must be >= 1, got %d", concurrency)
	}
	return &Hasher{sem: make(chan struct{}, concurrency)}, nil
}

func defaultHashConcurrency() int {
	return runtime.GOMAXPROCS(0)
}

func HashConcurrencyFromEnv() (int, error) {
	value := os.Getenv("AUTH_HASH_CONCURRENCY")
	if value == "" {
		return defaultHashConcurrency(), nil
	}

	concurrency, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid AUTH_HASH_CONCURRENCY %q: %w", value, err)
	}
	if concurrency < 1 {
		return 0, fmt.Errorf("invalid AUTH_HASH_CONCURRENCY %q: muss positiv sein", value)
	}
	return concurrency, nil
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-h.sem }()

	return hashPassword(password)
}

func (h *Hasher) Verify(ctx context.Context, encodedHash, password string) error {
	if err := h.acquire(ctx); err != nil {
		return err
	}
	defer func() { <-h.sem }()

	return verifyPassword(encodedHash, password)
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonParallelism, argonKeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonParallelism, b64Salt, b64Key), nil
}

func verifyPassword(encodedHash, password string) error {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return ErrInvalidHash
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ErrInvalidHash
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ErrInvalidHash
	}
	if len(salt) == 0 || len(hash) == 0 {
		return ErrInvalidHash
	}
	computed := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hash)))
	if subtle.ConstantTimeCompare(hash, computed) == 1 {
		return nil
	}
	return ErrMismatchedPassword
}
