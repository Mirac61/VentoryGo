package pdf

import (
	_ "embed"
	"fmt"

	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/fontrepository"
)

// The PDF core fonts (Helvetica/Arial) are never embedded, so a viewer
// without them substitutes a font that can mangle umlauts and the euro
// sign. Liberation Sans is metric-compatible with Arial and embeds cleanly.
const fontFamily = "LiberationSans"

//go:embed fonts/LiberationSans-Regular.ttf
var fontRegular []byte

//go:embed fonts/LiberationSans-Bold.ttf
var fontBold []byte

//go:embed fonts/LiberationSans-Italic.ttf
var fontItalic []byte

//go:embed fonts/LiberationSans-BoldItalic.ttf
var fontBoldItalic []byte

func embeddedFonts() ([]entity.CustomFont, error) {
	fonts, err := fontrepository.New().
		AddUTF8FontFromBytes(fontFamily, fontstyle.Normal, fontRegular).
		AddUTF8FontFromBytes(fontFamily, fontstyle.Bold, fontBold).
		AddUTF8FontFromBytes(fontFamily, fontstyle.Italic, fontItalic).
		AddUTF8FontFromBytes(fontFamily, fontstyle.BoldItalic, fontBoldItalic).
		Load()
	if err != nil {
		return nil, fmt.Errorf("pdf: loading embedded font failed: %w", err)
	}
	return fonts, nil
}
