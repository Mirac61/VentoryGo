package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasherNeedsPositiveConcurrency(t *testing.T) {
	for _, concurrency := range []int{0, -1} {
		hasher, err := NewHasher(concurrency)
		assert.Error(t, err, "concurrency=%d", concurrency)
		assert.Nil(t, hasher, "concurrency=%d", concurrency)
	}
}

func TestHasherKeepsConcurrentWorkBelowLimit(t *testing.T) {
	const limit = 3
	hasher, err := NewHasher(limit)
	require.NoError(t, err)

	var inFlight, peak atomic.Int64
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, hasher.acquire(context.Background()))
			defer func() { <-hasher.sem }()

			current := inFlight.Add(1)
			for {
				max := peak.Load()
				if current <= max || peak.CompareAndSwap(max, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			inFlight.Add(-1)
		}()
	}
	wg.Wait()

	assert.LessOrEqual(t, peak.Load(), int64(limit))
}

func TestHasherRefusesWorkOnCancelledContext(t *testing.T) {
	hasher, err := NewHasher(4)
	require.NoError(t, err)

	// Bei freiem Platz sind beide Zweige des select bereit, und die Wahl faellt zufaellig.
	// Ein einzelner Durchlauf wuerde einen fehlenden Vorab-Check nur manchmal bemerken.
	for i := range 50 {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		assert.ErrorIs(t, hasher.acquire(ctx), context.Canceled, "acquire, Durchlauf %d", i)

		hash, err := hasher.Hash(ctx, "correct horse battery")
		assert.ErrorIs(t, err, context.Canceled, "Hash, Durchlauf %d", i)
		assert.Empty(t, hash, "Durchlauf %d", i)

		assert.ErrorIs(t, hasher.Verify(ctx, dummyHash(), "correct horse battery"),
			context.Canceled, "Verify, Durchlauf %d", i)

		require.Empty(t, hasher.sem, "abgebrochene Aufrufe duerfen keinen Platz behalten, Durchlauf %d", i)
	}
}

func TestHashAndVerifyGoThroughTheSemaphore(t *testing.T) {
	hasher, err := NewHasher(1)
	require.NoError(t, err)
	hasher.sem <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	hash, err := hasher.Hash(ctx, "correct horse battery")
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	assert.Empty(t, hash)
	assert.True(t, errors.Is(hasher.Verify(ctx, dummyHash(), "correct horse battery"), context.DeadlineExceeded))

	<-hasher.sem
	hash, err = hasher.Hash(context.Background(), "correct horse battery")
	require.NoError(t, err)
	assert.NoError(t, hasher.Verify(context.Background(), hash, "correct horse battery"))
}

func TestHashPasswordFormat(t *testing.T) {
	hash, err := hashPassword("correct horse battery")
	require.NoError(t, err)

	// Ohne testify waere das: if !strings.HasPrefix(...) { t.Errorf(...) }
	assert.True(t, strings.HasPrefix(hash, "$argon2id$"), "hash %q ist kein PHC-String", hash)
}

func TestVerifyPasswordAcceptsCorrectPassword(t *testing.T) {
	const pw = "correct horse battery"

	hash, err := hashPassword(pw)
	require.NoError(t, err)

	assert.NoError(t, verifyPassword(hash, pw))
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := hashPassword("correct horse battery")
	require.NoError(t, err)

	assert.ErrorIs(t, verifyPassword(hash, "falsch"), ErrMismatchedPassword)
}

// Gleiches Passwort, zwei Aufrufe, zwei verschiedene Hashes: der Salt wirkt.
func TestHashPasswordUsesFreshSalt(t *testing.T) {
	const pw = "correct horse battery"

	first, err := hashPassword(pw)
	require.NoError(t, err)
	second, err := hashPassword(pw)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.NoError(t, verifyPassword(second, pw))
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, hash := range []string{
		"",
		"kein-phc-string",
		"$bcrypt$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$",
		"$argon2id$v=19$m=19456,t=2,p=1$$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$$",
	} {
		assert.ErrorIs(t, verifyPassword(hash, "egal"), ErrInvalidHash, "hash=%q", hash)
	}
}
