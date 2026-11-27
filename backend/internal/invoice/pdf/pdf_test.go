package pdf

import (
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"

	"bytes"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/johnfercher/go-tree/node"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/phpdave11/gofpdf"
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

var pageObject = regexp.MustCompile(`/Type /Page\b[^s]`)

func pageCount(doc []byte) int {
	return len(pageObject.FindAll(doc, -1))
}

func invoiceWithItems(n int) invoice.Invoice {
	inv := invoice.Invoice{Currency: "EUR"}
	inv.Sender.Name = "Ventory Handels GmbH"
	inv.Sender.IBAN = "DE89370400440532013000"
	for i := 1; i <= n; i++ {
		inv.Items = append(inv.Items, invoice.LineItem{
			Position: i, Description: "Wartungspauschale Modul " + strconv.Itoa(i),
			Quantity: 1000, Unit: "Stk.", UnitPrice: 1000, Total: 1000, VatRate: 1900,
		})
	}
	return inv
}

func TestGenerate_Pagination(t *testing.T) {
	t.Run("a short invoice fits on one page", func(t *testing.T) {
		doc, err := Generate(invoiceWithItems(3), Modern)

		require.NoError(t, err)
		assert.Equal(t, 1, pageCount(doc))
	})

	t.Run("a long invoice continues on further pages", func(t *testing.T) {
		doc, err := Generate(invoiceWithItems(60), Modern)

		require.NoError(t, err)
		assert.Greater(t, pageCount(doc), 1)
	})

	t.Run("every item count around the page break renders", func(t *testing.T) {
		for _, design := range []Design{Classic, Modern, Minimal} {
			for n := 15; n <= 40; n++ {
				_, err := Generate(invoiceWithItems(n), design)
				require.NoError(t, err, "n=%d", n)
			}
		}
	})

	t.Run("a single item with a description longer than a page renders", func(t *testing.T) {
		inv := invoiceWithItems(1)
		inv.Items[0].Description = strings.Repeat("Lorem ipsum dolor sit amet ", 400)

		doc, err := Generate(inv, Modern)

		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(doc, []byte("%PDF-")))
	})
}

func pagesTexts(t *testing.T, testInvoice invoice.Invoice, design Design) [][]string {
	document, err := buildDocument(testInvoice, design)
	require.NoError(t, err)

	var collectTexts func(structureNode *node.Node[core.Structure]) []string
	collectTexts = func(structureNode *node.Node[core.Structure]) []string {
		var texts []string
		if text, isText := structureNode.GetData().Value.(string); isText {
			texts = append(texts, text)
		}
		for _, child := range structureNode.GetNexts() {
			texts = append(texts, collectTexts(child)...)
		}
		return texts
	}

	var pages [][]string
	for _, pageNode := range document.GetStructure().GetNexts() {
		pages = append(pages, collectTexts(pageNode))
	}
	return pages
}

// totalsBlockPageTexts is the text of the page that carries the totals block.
func totalsBlockPageTexts(t *testing.T, testInvoice invoice.Invoice, design Design) []string {
	for _, pageTexts := range pagesTexts(t, testInvoice, design) {
		if slices.Contains(pageTexts, "Nettobetrag") {
			return pageTexts
		}
	}
	require.Fail(t, "no page carries the totals block")
	return nil
}

// invoiceFromService runs the items through the real service, so the PDF is
// checked against the totals the API itself returns.
func invoiceFromService(t *testing.T, items ...invoice.LineItem) invoice.Invoice {
	draft := invoiceWithItems(0)
	draft.Items = items
	created, err := invoice.NewService(invoice.NewRepository()).Create(draft, "owner")
	require.NoError(t, err)
	return created
}

func lineItem(description string, unitPriceInCents int64, vatRateInBasisPoints int) invoice.LineItem {
	return invoice.LineItem{
		Description: description,
		Quantity:    1000,
		Unit:        "Stk.",
		UnitPrice:   invoice.Money(unitPriceInCents),
		VatRate:     vatRateInBasisPoints,
	}
}

func TestGenerate_TotalsBlock(t *testing.T) {
	mixedRateInvoice := func(itemCount int) invoice.Invoice {
		mixed := invoiceWithItems(itemCount)
		mixed.Items[0].VatRate, mixed.Items[0].Total = 700, 20000
		mixed.Items[1%itemCount].VatRate = 1900
		mixed.VatBreakdown = []invoice.VATBreakdownEntry{
			{VatRate: 700, NetAmount: 20000, VatAmount: 1400},
			{VatRate: 1900, NetAmount: 5000, VatAmount: 950},
		}
		mixed.NetTotal, mixed.VATAmount, mixed.GrossTotal = 25000, 2350, 27350
		return mixed
	}

	t.Run("one line per rate in API order, net and gross around them", func(t *testing.T) {
		blockText := strings.Join(totalsBlockPageTexts(t, mixedRateInvoice(3), Modern), "|")

		expectedOrder := []string{"Nettobetrag", "zzgl. 7 % USt auf 200,00", "14,00", "zzgl. 19 % USt auf 50,00", "9,50", "Gesamtbetrag", "273,50"}
		previousPosition := -1
		for _, expectedText := range expectedOrder {
			position := strings.Index(blockText, expectedText)
			require.Greater(t, position, previousPosition, "%q out of order or missing", expectedText)
			previousPosition = position
		}
	})

	t.Run("a single rate still gets its breakdown line", func(t *testing.T) {
		singleRateInvoice := invoiceWithItems(2)
		singleRateInvoice.VatBreakdown = []invoice.VATBreakdownEntry{{VatRate: 1900, NetAmount: 2000, VatAmount: 380}}

		assert.Contains(t, totalsBlockPageTexts(t, singleRateInvoice, Modern), "zzgl. 19 % USt auf 20,00")
	})

	t.Run("the block is never split across pages", func(t *testing.T) {
		for _, design := range []Design{Classic, Modern, Minimal} {
			blockMovedToLaterPage := false
			for itemCount := 1; itemCount <= 45; itemCount++ {
				for pageIndex, pageTexts := range pagesTexts(t, mixedRateInvoice(itemCount), design) {
					pageContains := func(text string) bool { return slices.Contains(pageTexts, text) }
					assert.Equal(t, pageContains("Nettobetrag"), pageContains("Gesamtbetrag"), "items=%d page %d", itemCount, pageIndex+1)
					assert.Equal(t, pageContains("Nettobetrag"), pageContains("zzgl. 19 % USt auf 50,00"), "items=%d page %d", itemCount, pageIndex+1)
					// A later page with so little text holds the block but no item table.
					blockMovedToLaterPage = blockMovedToLaterPage || (pageContains("Nettobetrag") && pageIndex > 0 && len(pageTexts) < 40)
				}
			}
			assert.True(t, blockMovedToLaterPage, "sweep never hit a page break before the block")
		}
	})

	t.Run("a vat-exempt invoice shows no tax lines but stays whole", func(t *testing.T) {
		for itemCount := 1; itemCount <= 45; itemCount++ {
			exemptInvoice := invoiceWithItems(itemCount)
			exemptInvoice.VatExempt = true
			exemptInvoice.NetTotal, exemptInvoice.GrossTotal = 1000*invoice.Money(itemCount), 1000*invoice.Money(itemCount)

			pageTexts := totalsBlockPageTexts(t, exemptInvoice, Modern)

			assert.Contains(t, pageTexts, "Gesamtbetrag", "items=%d", itemCount)
			assert.NotContains(t, strings.Join(pageTexts, "|"), "USt auf", "items=%d", itemCount)
		}
	})
}

func TestGenerate_TotalsMatchAPI(t *testing.T) {
	// 3 x 0,33 at 19 % rounds 18,81 ct up to 19 ct; the PDF must not re-round.
	apiInvoice := invoiceFromService(t,
		lineItem("Kabel", 33, 1900), lineItem("Stecker", 33, 1900), lineItem("Adapter", 33, 1900),
		lineItem("Fachbuch", 2000, 700),
	)
	require.Len(t, apiInvoice.VatBreakdown, 2)

	pageTexts := totalsBlockPageTexts(t, apiInvoice, Modern)

	// Hard-coded, so a rounding change in either place fails the test.
	for _, expectedText := range []string{
		"20,99 €",
		"zzgl. 7 % USt auf 20,00", "1,40 €",
		"zzgl. 19 % USt auf 0,99", "0,19 €",
		"22,58 €",
	} {
		assert.Contains(t, pageTexts, expectedText)
	}
	// Every API figure appears verbatim.
	assert.Contains(t, pageTexts, formatMoney(apiInvoice.NetTotal, apiInvoice.Currency))
	assert.Contains(t, pageTexts, formatMoney(apiInvoice.GrossTotal, apiInvoice.Currency))
	for _, entry := range apiInvoice.VatBreakdown {
		assert.Contains(t, pageTexts, formatMoney(entry.VatAmount, apiInvoice.Currency))
		assert.Contains(t, pageTexts, "zzgl. "+formatVatRate(entry.VatRate)+" USt auf "+formatAmount(entry.NetAmount))
	}
}

func TestVatRowsOf(t *testing.T) {
	t.Run("keeps the API order", func(t *testing.T) {
		vatRows := vatRowsOf(invoice.Invoice{Currency: "EUR", VatBreakdown: []invoice.VATBreakdownEntry{
			{VatRate: 1900, NetAmount: 100, VatAmount: 19},
			{VatRate: 700, NetAmount: 100, VatAmount: 7},
		}})

		assert.Equal(t, [][2]string{
			{"zzgl. 19 % USt auf 1,00", "0,19 €"},
			{"zzgl. 7 % USt auf 1,00", "0,07 €"},
		}, vatRows)
	})

	t.Run("fractional rate and thousands separator", func(t *testing.T) {
		vatRows := vatRowsOf(invoice.Invoice{Currency: "EUR", VatBreakdown: []invoice.VATBreakdownEntry{
			{VatRate: 550, NetAmount: 123456789, VatAmount: 6790123},
		}})

		assert.Equal(t, [][2]string{{"zzgl. 5,5 % USt auf 1.234.567,89", "67.901,23 €"}}, vatRows)
	})

	t.Run("without breakdown falls back to the single vat amount", func(t *testing.T) {
		vatRows := vatRowsOf(invoice.Invoice{Currency: "EUR", VATAmount: 380})

		assert.Equal(t, [][2]string{{"zzgl. Umsatzsteuer", "3,80 €"}}, vatRows)
	})
}

func TestPriceHeading(t *testing.T) {
	assert.Equal(t, "Einzelpreis (EUR)", priceHeading("EUR"))
	assert.Equal(t, "Einzelpreis (USD)", priceHeading("USD"))
	assert.Equal(t, "Einzelpreis", priceHeading(""))
}

func TestColumnGroups_FillGrid(t *testing.T) {
	assert.Equal(t, gridSize, colAddress+colGutter+colInfo)
	assert.Equal(t, gridSize, colPos+colDesc+colQty+colPrice+colVat+colAmount)
}

func TestItemTableHeader_LabelsFitColumns(t *testing.T) {
	contentWidth := 210.0 - pageLeftMargin - pageRightMargin
	fpdf := gofpdf.New("P", "mm", "A4", "")
	fpdf.AddUTF8FontFromBytes(fontFamily, "B", fontBold)

	for name, design := range map[string]Design{"classic": Classic, "modern": Modern, "minimal": Minimal} {
		fpdf.SetFont(fontFamily, "B", design.sizeSmall)
		for _, c := range itemColumns("EUR") {
			label := design.headLabel(c.label)
			// At least one space between neighbouring labels.
			needed := fpdf.GetStringWidth(label + " ")
			available := contentWidth * float64(c.span) / gridSize
			assert.LessOrEqual(t, needed, available, "%s: %q", name, label)
		}
	}
}
