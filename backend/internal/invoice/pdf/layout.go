package pdf

import (
	"github.com/Mirac61/VentoryGo/backend/internal/invoice"

	"bytes"
	"fmt"
	"strconv"

	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// DIN 5008 Form B page geometry, in mm.
const (
	pageLeftMargin   = 25.0
	pageRightMargin  = 20.0
	pageTopMargin    = 15.0
	pageBottomMargin = 12.0

	addressFieldTop    = 45.0
	addressFieldHeight = 45.0

	// Finer grid so the six item columns fit the content width.
	gridSize = 24
)

// Each group must sum to gridSize.
const (
	colAddress = 12
	colGutter  = 2
	colInfo    = 10

	colPos    = 1
	colDesc   = 9
	colQty    = 3
	colPrice  = 4
	colVat    = 3
	colAmount = 4
)

const lineHeight = 4.2

// Height of the row that carries the letterhead rule.
const letterheadRuleRow = 2.0

func addLetterhead(m core.Maroto, inv invoice.Invoice, design Design) {
	nameAlign := align.Left
	if !design.logoOnRight {
		nameAlign = align.Right
	}
	nameCol := col.New(gridSize/2).Add(
		text.New(inv.Sender.Name, props.Text{
			Size:  design.sizeCompany,
			Style: fontstyle.Bold,
			Color: design.accent,
			Align: nameAlign,
			Top:   1,
		}),
		text.New(senderContactLine(inv.Sender), props.Text{
			Size:  design.sizeMicro,
			Color: design.muted,
			Align: nameAlign,
			Top:   design.sizeCompany*0.42 + 2.5,
		}),
	)

	logoCol := logoColumn(inv.Sender.Logo, gridSize/2, design)

	if design.logoOnRight {
		m.AddRow(design.letterheadHeight, nameCol, logoCol)
	} else {
		m.AddRow(design.letterheadHeight, logoCol, nameCol)
	}

	if design.letterheadRule > 0 {
		m.AddRow(letterheadRuleRow, col.New(gridSize).WithStyle(&props.Cell{
			BorderType:      border.Bottom,
			BorderColor:     design.accent,
			BorderThickness: design.letterheadRule,
		}))
	}
}

func logoColumn(logo []byte, size int, design Design) core.Col {
	ext, ok := imageExtension(logo)
	if !ok {
		return col.New(size)
	}
	// maroto images can't align; Center approximates right anchoring.
	return image.NewFromBytesCol(size, logo, ext, props.Rect{
		Percent: design.logoScale,
		Center:  design.logoOnRight,
	})
}

func imageExtension(data []byte) (extension.Type, bool) {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return extension.Png, true
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return extension.Jpg, true
	default:
		return "", false
	}
}

func addAddressAndInfoBlock(m core.Maroto, inv invoice.Invoice, design Design) {
	headroom := design.letterheadHeight
	if design.letterheadRule > 0 {
		headroom += letterheadRuleRow
	}
	m.AddRows(spacer(addressFieldTop - pageTopMargin - headroom))

	addressCol := col.New(colAddress).Add(
		text.New(senderOneLiner(inv.Sender), props.Text{
			Size:  design.sizeMicro,
			Color: design.muted,
			Style: fontstyle.Underline,
			Top:   1,
		}),
		text.New(inv.Recipient.Name, props.Text{
			Size:  design.sizeBody,
			Style: fontstyle.Bold,
			Color: design.ink,
			Top:   12.7, // end of the notice zone
		}),
	)
	for i, addressLine := range recipientLines(inv.Recipient) {
		addressCol.Add(text.New(addressLine, props.Text{
			Size:  design.sizeBody,
			Color: design.ink,
			Top:   12.7 + float64(i+1)*lineHeight,
		}))
	}

	infoRows := infoBlockRows(inv)
	infoCol := col.New(colInfo)
	for i, pair := range infoRows {
		top := 3 + float64(i)*(lineHeight+0.6)
		infoCol.Add(
			text.New(pair[0], props.Text{
				Size: design.sizeSmall, Color: design.muted, Top: top, Left: 3,
			}),
			text.New(pair[1], props.Text{
				Size: design.sizeSmall, Style: fontstyle.Bold, Color: design.ink,
				Top: top, Align: align.Right, Right: 3,
			}),
		)
	}
	if design.infoPanelBg != nil {
		infoCol.WithStyle(&props.Cell{BackgroundColor: design.infoPanelBg})
	}

	addressHeight := 12.7 + float64(len(recipientLines(inv.Recipient))+1)*lineHeight
	infoHeight := 6 + float64(len(infoRows))*(lineHeight+0.6)
	blockHeight := addressHeight
	if infoHeight > blockHeight {
		blockHeight = infoHeight
	}

	m.AddRow(blockHeight, addressCol, col.New(colGutter), infoCol)
	m.AddRows(spacer(addressFieldHeight - blockHeight))
}

func addSubject(m core.Maroto, inv invoice.Invoice, design Design) {
	subject := "Rechnungsentwurf"
	if inv.InvoiceNumber != nil {
		subject = "Rechnung " + *inv.InvoiceNumber
	}
	m.AddRow(9, text.NewCol(gridSize, subject, props.Text{
		Size:  design.sizeSubject,
		Style: fontstyle.Bold,
		Color: design.accent,
		Top:   1,
	}))
	m.AddRow(7, text.NewCol(gridSize, serviceSentence(inv), props.Text{
		Size:  design.sizeSmall,
		Color: design.muted,
	}))
}

func addItemsTable(m core.Maroto, inv invoice.Invoice, design Design) {
	m.AddRows(spacer(4))

	label := func(s string) string {
		if design.tableHeadUpper {
			return upper(s)
		}
		return s
	}
	head := props.Text{Size: design.sizeSmall, Style: fontstyle.Bold, Color: design.ink, Top: 2}
	if design.tableHeadInk != nil {
		head.Color = design.tableHeadInk
	}
	headRight := head
	headRight.Align = align.Right

	headerRow := m.AddRow(design.tableRowSpacing+1,
		text.NewCol(colPos, label("Pos."), head),
		text.NewCol(colDesc, label("Bezeichnung"), head),
		text.NewCol(colQty, label("Menge"), headRight),
		text.NewCol(colPrice, label("Einzelpreis"), headRight),
		text.NewCol(colVat, label("Steuer"), headRight),
		text.NewCol(colAmount, label("Betrag"), headRight),
	)
	headerStyle := &props.Cell{BackgroundColor: design.tableHeadBg}
	if design.tableHeadRule {
		headerStyle.BorderType = border.Bottom
		headerStyle.BorderColor = design.accent
		headerStyle.BorderThickness = 0.4
	}
	headerRow.WithStyle(headerStyle)

	body := props.Text{Size: design.sizeBody, Color: design.ink, Top: 2}
	bodyRight := body
	bodyRight.Align = align.Right

	for i, item := range inv.Items {
		itemRow := m.AddAutoRow(
			text.NewCol(colPos, strconv.Itoa(i+1), body),
			text.NewCol(colDesc, item.Description, body),
			text.NewCol(colQty, formatQuantityWithUnit(item.Quantity, item.Unit), bodyRight),
			text.NewCol(colPrice, formatAmount(item.UnitPrice), bodyRight),
			text.NewCol(colVat, formatVatRate(item.VatRate), bodyRight),
			text.NewCol(colAmount, formatAmount(item.Total), bodyRight),
		)
		switch {
		case design.zebraBg != nil && i%2 == 1:
			itemRow.WithStyle(&props.Cell{BackgroundColor: design.zebraBg})
		case design.tableRowRule:
			itemRow.WithStyle(&props.Cell{
				BorderType:      border.Bottom,
				BorderColor:     design.hairline,
				BorderThickness: 0.1,
			})
		}
	}
}

func addTotals(m core.Maroto, inv invoice.Invoice, design Design) {
	m.AddRows(spacer(3))

	const (
		totalsIndent = colPos + colDesc
		totalsLabel  = colQty + colPrice
		totalsValue  = colVat + colAmount
	)

	subtotal := func(label, value string) {
		m.AddRow(6,
			col.New(totalsIndent),
			text.NewCol(totalsLabel, label, props.Text{
				Size: design.sizeSmall, Color: design.muted, Align: align.Right, Top: 1.4,
			}),
			text.NewCol(totalsValue, value, props.Text{
				Size: design.sizeBody, Color: design.ink, Align: align.Right, Top: 1.2, Right: 2,
			}),
		)
	}

	subtotal("Nettobetrag", formatMoney(inv.NetTotal, inv.Currency))
	if !inv.VatExempt {
		for _, entry := range vatRowsOf(inv) {
			subtotal(entry[0], entry[1])
		}
	}

	if design.totalRule > 0 {
		m.AddRow(3,
			col.New(totalsIndent),
			col.New(totalsLabel+totalsValue).WithStyle(&props.Cell{
				BorderType:      border.Top,
				BorderColor:     design.accent,
				BorderThickness: design.totalRule,
			}),
		)
	}

	const totalRowHeight = 13
	totalColour := design.totalInk
	if design.totalInkIsAccent {
		totalColour = design.accent
	}
	totalRow := m.AddRow(totalRowHeight,
		col.New(totalsIndent),
		text.NewCol(totalsLabel, "Gesamtbetrag", props.Text{
			Size: design.sizeSmall, Style: fontstyle.Bold, Color: totalColour,
			Align: align.Right, Top: centerTop(totalRowHeight, design.sizeSmall), Left: 3,
		}),
		text.NewCol(totalsValue, formatMoney(inv.GrossTotal, inv.Currency), props.Text{
			Size: design.sizeTotal, Style: fontstyle.Bold, Color: totalColour,
			Align: align.Right, Top: centerTop(totalRowHeight, design.sizeTotal), Right: 3,
		}),
	)
	if design.totalBg != nil || design.totalBorder != border.None {
		totalRow.WithStyle(&props.Cell{
			BackgroundColor: design.totalBg,
			BorderType:      design.totalBorder,
			BorderColor:     design.ink,
			BorderThickness: 0.3,
		})
	}
}

func addPaymentBlock(m core.Maroto, inv invoice.Invoice, design Design) {
	if inv.Sender.IBAN == "" {
		return
	}
	m.AddRows(spacer(8))

	m.AddRow(6, text.NewCol(gridSize, upper("Zahlungsbedingungen"), props.Text{
		Size: design.sizeMicro, Style: fontstyle.Bold, Color: design.muted,
	}))
	m.AddRow(3, col.New(gridSize).WithStyle(&props.Cell{
		BorderType:      border.Top,
		BorderColor:     design.hairline,
		BorderThickness: 0.2,
	}))

	due := fmt.Sprintf("Bitte überweisen Sie %s bis zum %s auf folgendes Konto:",
		formatMoney(inv.GrossTotal, inv.Currency), inv.PaymentDueAt.Format("02.01.2006"))
	m.AddRow(6, text.NewCol(gridSize, due, props.Text{
		Size: design.sizeSmall, Color: design.ink, Top: 1,
	}))

	bank := [][2]string{{"IBAN", inv.Sender.IBAN}}
	if inv.Sender.BIC != "" {
		bank = append(bank, [2]string{"BIC", inv.Sender.BIC})
	}
	if inv.Sender.BankName != "" {
		bank = append(bank, [2]string{"Bank", inv.Sender.BankName})
	}
	if inv.InvoiceNumber != nil {
		bank = append(bank, [2]string{"Verwendungszweck", *inv.InvoiceNumber})
	}
	for _, pair := range bank {
		m.AddRow(4.6,
			text.NewCol(6, pair[0], props.Text{Size: design.sizeSmall, Color: design.muted}),
			text.NewCol(gridSize-6, pair[1], props.Text{
				Size: design.sizeSmall, Style: fontstyle.Bold, Color: design.ink,
			}),
		)
	}
}

func addNotesAndLegalNotices(m core.Maroto, inv invoice.Invoice, design Design) {
	if inv.Notes != "" {
		m.AddRows(spacer(7))
		m.AddAutoRow(text.NewCol(gridSize, inv.Notes, props.Text{
			Size: design.sizeSmall, Color: design.ink,
		}))
	}
	if len(inv.LegalNotices) > 0 {
		m.AddRows(spacer(4))
		for _, notice := range inv.LegalNotices {
			m.AddAutoRow(text.NewCol(gridSize, notice, props.Text{
				Size: design.sizeMicro, Style: fontstyle.Italic, Color: design.muted,
			}))
		}
	}
}

func footerRows(inv invoice.Invoice, design Design) []core.Row {
	micro := props.Text{Size: design.sizeMicro, Color: design.muted}

	legal := []string{inv.Sender.Name, inv.Sender.Street, fmt.Sprintf("%s %s", inv.Sender.Zip, inv.Sender.City)}

	tax := []string{}
	if inv.Sender.VatID != "" {
		tax = append(tax, "USt-IdNr. "+inv.Sender.VatID)
	}
	if inv.Sender.TaxNumber != "" {
		tax = append(tax, "Steuernr. "+inv.Sender.TaxNumber)
	}
	if inv.Sender.Email != "" {
		tax = append(tax, inv.Sender.Email)
	}

	bank := []string{}
	if inv.Sender.BankName != "" {
		bank = append(bank, inv.Sender.BankName)
	}
	if inv.Sender.IBAN != "" {
		bank = append(bank, inv.Sender.IBAN)
	}
	if inv.Sender.BIC != "" {
		bank = append(bank, inv.Sender.BIC)
	}

	stack := func(size int, lines []string, alignment align.Type) core.Col {
		c := col.New(size)
		for i, l := range lines {
			p := micro
			p.Top = 2 + float64(i)*3
			p.Align = alignment
			c.Add(text.New(l, p))
		}
		return c
	}

	rule := footerRule(design)
	body := row.New(14).Add(
		stack(8, legal, align.Left),
		stack(8, tax, align.Left),
		stack(8, bank, align.Right),
	)
	return []core.Row{rule, body}
}

func footerRule(design Design) core.Row {
	colour := design.hairline
	if design.footerRuleIsAccent {
		colour = design.accent
	}
	return row.New(2).Add(col.New(gridSize).WithStyle(&props.Cell{
		BorderType:      border.Top,
		BorderColor:     colour,
		BorderThickness: 0.2,
	}))
}

// 1 pt is 0.353 mm.
func centerTop(rowHeight, fontSize float64) float64 {
	top := (rowHeight - fontSize*0.353) / 2
	if top < 0 {
		return 0
	}
	return top
}

func spacer(height float64) core.Row {
	if height < 0 {
		height = 0
	}
	return row.New(height)
}
