package pdf

import (
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"

	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatMoney_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		amount   invoice.Money
		currency string
		want     string
	}{
		{name: "zero", amount: 0, currency: "EUR", want: "0,00 €"},
		{name: "negative amount below one unit keeps its sign", amount: -50, currency: "EUR", want: "-0,50 €"},
		{name: "negative amount at exactly one unit", amount: -100, currency: "EUR", want: "-1,00 €"},
		{name: "negative single cent keeps its sign", amount: -1, currency: "EUR", want: "-0,01 €"},
		{name: "large amount uses thousands separator", amount: 123456789, currency: "EUR", want: "1.234.567,89 €"},
		{name: "empty currency", amount: 100, currency: "", want: "1,00 "},
		{name: "non-euro currency keeps its ISO code", amount: 100, currency: "USD", want: "1,00 USD"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, formatMoney(test.amount, test.currency))
		})
	}
}

func TestGenerate_EdgeCases(t *testing.T) {
	base := invoice.Invoice{
		Currency:   "EUR",
		GrossTotal: 0,
	}

	t.Run("no items does not error", func(t *testing.T) {
		inv := base
		inv.Items = nil

		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})

	t.Run("nil invoice number omits the number line without erroring", func(t *testing.T) {
		inv := base
		inv.InvoiceNumber = nil
		inv.Items = []invoice.LineItem{{Description: "X", Total: 100}}

		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})

	t.Run("negative gross total (credit note) does not error", func(t *testing.T) {
		inv := base
		inv.Items = []invoice.LineItem{{Description: "Gutschrift", Total: -5000}}
		inv.GrossTotal = -5000

		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})

	t.Run("very long description does not error", func(t *testing.T) {
		inv := base
		inv.Items = []invoice.LineItem{{Description: strings.Repeat("Lorem ipsum dolor sit amet ", 200), Total: 100}}

		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})

	t.Run("large number of items does not error", func(t *testing.T) {
		inv := base
		inv.Items = make([]invoice.LineItem, 500)
		for i := range inv.Items {
			inv.Items[i] = invoice.LineItem{Description: "Position", Total: 100}
		}

		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})

	t.Run("VAT exempt without legal notices still renders (auto-populated notice)", func(t *testing.T) {
		inv := base
		inv.Items = []invoice.LineItem{{Description: "X", Total: 100}}
		inv.VatExempt = true
		inv.LegalNotices = nil

		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})
}

func TestParseHexColor_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *props.Color
	}{
		{name: "empty falls back to the design default", value: "", want: nil},
		{name: "without leading hash", value: "3E5880", want: &props.Color{Red: 62, Green: 88, Blue: 128}},
		{name: "lowercase digits", value: "#3e5880", want: &props.Color{Red: 62, Green: 88, Blue: 128}},
		{name: "three-digit shorthand is not supported", value: "#abc", want: nil},
		{name: "eight digits with alpha", value: "#3E5880FF", want: nil},
		{name: "non-hex characters", value: "#GGGGGG", want: nil},
		{name: "hash only", value: "#", want: nil},
		{name: "black", value: "#000000", want: &props.Color{Red: 0, Green: 0, Blue: 0}},
		{name: "white", value: "#FFFFFF", want: &props.Color{Red: 255, Green: 255, Blue: 255}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, parseHexColor(test.value))
		})
	}
}

func TestTint_EdgeCases(t *testing.T) {
	navy := &props.Color{Red: 0, Green: 50, Blue: 100}

	t.Run("full opacity returns the colour untouched", func(t *testing.T) {
		assert.Equal(t, navy, tint(navy, 1))
	})

	t.Run("zero opacity returns white", func(t *testing.T) {
		assert.Equal(t, &props.Color{Red: 255, Green: 255, Blue: 255}, tint(navy, 0))
	})

	t.Run("opacity above one is clamped instead of overshooting", func(t *testing.T) {
		assert.Equal(t, navy, tint(navy, 2.5))
	})

	t.Run("white stays white at any opacity", func(t *testing.T) {
		white := &props.Color{Red: 255, Green: 255, Blue: 255}
		assert.Equal(t, white, tint(white, 0.3))
	})
}

func TestReadableInk_EdgeCases(t *testing.T) {
	dark := &props.Color{Red: 17, Green: 24, Blue: 39}
	white := &props.Color{Red: 255, Green: 255, Blue: 255}

	t.Run("no background keeps the dark ink", func(t *testing.T) {
		assert.Equal(t, dark, readableInk(nil, dark))
	})

	t.Run("near-black background flips to white", func(t *testing.T) {
		assert.Equal(t, white, readableInk(&props.Color{Red: 10, Green: 10, Blue: 10}, dark))
	})

	t.Run("pale background keeps the dark ink", func(t *testing.T) {
		assert.Equal(t, dark, readableInk(&props.Color{Red: 240, Green: 244, Blue: 250}, dark))
	})

	t.Run("saturated green is bright enough for dark ink", func(t *testing.T) {
		assert.Equal(t, dark, readableInk(&props.Color{Red: 0, Green: 220, Blue: 0}, dark))
	})
}

func TestGenerate_BrandColour(t *testing.T) {
	inv := invoice.Invoice{Currency: "EUR", Items: []invoice.LineItem{{Description: "X", Total: 100}}}

	t.Run("invalid brand colour still renders", func(t *testing.T) {
		inv.Sender.AccentColor = "not-a-colour"

		doc, err := Generate(inv, Modern)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})

	t.Run("brand colour changes the rendered bytes", func(t *testing.T) {
		inv.Sender.AccentColor = ""
		plain, err := Generate(inv, Modern)
		require.NoError(t, err)

		inv.Sender.AccentColor = "#7A3B4A"
		branded, err := Generate(inv, Modern)
		require.NoError(t, err)

		assert.NotEqual(t, plain, branded, "the accent colour must reach the page")
	})
}

func TestGenerate_CreationDate(t *testing.T) {
	inv := invoice.Invoice{
		Currency:  "EUR",
		Items:     []invoice.LineItem{{Description: "X", Total: 100}},
		CreatedAt: time.Date(2026, 1, 15, 10, 30, 45, 0, time.UTC),
	}

	t.Run("the invoice's CreatedAt reaches the PDF's /CreationDate, not time.Now", func(t *testing.T) {
		doc, err := Generate(inv, Default)

		require.NoError(t, err)
		assert.Contains(t, string(doc), "D:20260115103045", "the PDF metadata must carry the caller-supplied timestamp")
	})

	t.Run("a different CreatedAt changes the metadata accordingly", func(t *testing.T) {
		other := inv
		other.CreatedAt = inv.CreatedAt.Add(24 * time.Hour)

		doc, err := Generate(other, Default)

		require.NoError(t, err)
		assert.Contains(t, string(doc), "D:20260116103045")
	})
}
