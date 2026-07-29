package invoice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextCounter_ResetsOnNewYear(t *testing.T) {
	repo := NewRepository()

	first, err := repo.nextCounter(time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	second, err := repo.nextCounter(time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 2, second)

	third, err := repo.nextCounter(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, third)
}
