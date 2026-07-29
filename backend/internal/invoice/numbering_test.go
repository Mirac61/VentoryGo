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

func TestNumberingFromEnv_Defaults(t *testing.T) {
	// Empty rather than unset, so the test does not depend on the developer's
	// shell and still covers the fallback.
	t.Setenv("INVOICE_NUMBER_PREFIX", "")
	t.Setenv("INVOICE_TIMEZONE", "")

	numbering, err := NumberingFromEnv()

	require.NoError(t, err)
	assert.Equal(t, "INV", numbering.Prefix)
	assert.Equal(t, "Europe/Berlin", numbering.Location.String())
}

func TestNumberingFromEnv_OverridesPrefixAndZone(t *testing.T) {
	t.Setenv("INVOICE_NUMBER_PREFIX", "RE")
	t.Setenv("INVOICE_TIMEZONE", "Pacific/Auckland")

	numbering, err := NumberingFromEnv()

	require.NoError(t, err)
	assert.Equal(t, "RE-2026-0007", numbering.Format(2026, 7))

	// Auckland is a day ahead of Berlin, which is the point of making the zone
	// configurable at all.
	newYear := time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, 2026, newYear.In(numbering.Location).Year())
}

func TestNumberingFromEnv_UnknownZoneFails(t *testing.T) {
	t.Setenv("INVOICE_TIMEZONE", "Europe/Nirgendwo")

	_, err := NumberingFromEnv()

	require.Error(t, err)
}
