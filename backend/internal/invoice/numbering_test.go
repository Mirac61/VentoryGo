package invoice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNumberingFormat(t *testing.T) {
	numbering := DefaultNumbering()

	assert.Equal(t, "INV-2026-0001", numbering.Format(2026, 1))
	assert.Equal(t, "INV-2026-0042", numbering.Format(2026, 42))
	assert.Equal(t, "INV-2026-10000", numbering.Format(2026, 10000), "a year with more than 9999 invoices keeps counting")
}

func TestDefaultNumbering(t *testing.T) {
	numbering := DefaultNumbering()

	assert.Equal(t, "INV", numbering.Prefix)
	assert.Equal(t, "Europe/Berlin", numbering.Location.String(),
		"must match the column defaults on users")
}

func TestNewNumbering_TakesPrefixAndZone(t *testing.T) {
	numbering, err := NewNumbering("RE", "Pacific/Auckland")

	require.NoError(t, err)
	assert.Equal(t, "RE-2026-0007", numbering.Format(2026, 7))

	newYear := time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, 2026, newYear.In(numbering.Location).Year(), "Auckland is a day ahead of Berlin")
}

// The zone is a tenant value from users, so a bad one must not kill the process.
func TestNewNumbering_UnknownZoneFailsWithoutPanic(t *testing.T) {
	require.NotPanics(t, func() {
		_, err := NewNumbering("INV", "Europe/Nirgendwo")
		require.Error(t, err)
	})
}
