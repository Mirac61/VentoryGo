package invoice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextCounter_ResetsOnNewYear(t *testing.T) {
	repo := NewRepository()

	first, err := repo.nextCounter(testOwner, time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	second, err := repo.nextCounter(testOwner, time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 2, second)

	third, err := repo.nextCounter(testOwner, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, third)
}

func TestNextCounter_RunsPerOwner(t *testing.T) {
	repo := NewRepository()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	for want := 1; want <= 3; want++ {
		for _, owner := range []string{testOwner, otherOwner} {
			counter, err := repo.nextCounter(owner, now)
			require.NoError(t, err)
			assert.Equalf(t, want, counter, "owner %s", owner)
		}
	}
}
