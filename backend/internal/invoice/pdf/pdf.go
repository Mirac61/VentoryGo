package pdf

import (
	"fmt"

	"github.com/Mirac61/VentoryGo/backend/internal/invoice"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

var Default = Classic

func Generate(inv invoice.Invoice, design Design) ([]byte, error) {
	// Zero Design renders a blank page; fall back to default.
	if design.sizeBody == 0 {
		design = Default
	}
	design = design.resolve(parseHexColor(inv.Sender.AccentColor))

	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(pageLeftMargin).
		WithRightMargin(pageRightMargin).
		WithTopMargin(pageTopMargin).
		WithBottomMargin(pageBottomMargin).
		WithMaxGridSize(gridSize).
		WithPageNumber(props.PageNumber{
			Pattern: "Seite {current} von {total}",
			Place:   props.RightBottom,
			Size:    design.sizeMicro,
			Color:   design.muted,
		}).
		Build()

	m := maroto.New(cfg)

	if err := m.RegisterFooter(footerRows(inv, design)...); err != nil {
		return nil, fmt.Errorf("pdf: footer setup failed: %w", err)
	}

	addLetterhead(m, inv, design)
	addAddressAndInfoBlock(m, inv, design)
	addSubject(m, inv, design)
	addItemsTable(m, inv, design)
	addTotals(m, inv, design)
	addPaymentBlock(m, inv, design)
	addNotesAndLegalNotices(m, inv, design)

	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf: generation failed: %w", err)
	}
	return document.GetBytes(), nil
}
