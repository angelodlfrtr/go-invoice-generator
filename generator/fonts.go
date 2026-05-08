package generator

import (
	_ "embed"

	"codeberg.org/go-pdf/fpdf"
)

//go:embed fonts/Roboto-Regular.ttf
var robotoRegular []byte

//go:embed fonts/Roboto-Bold.ttf
var robotoBold []byte

// registerDefaultFonts registers the embedded Roboto family on pdf so that
// the default Font / BoldFont options work without any external files.
func registerDefaultFonts(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes("Roboto", "", robotoRegular)
	pdf.AddUTF8FontFromBytes("Roboto", "B", robotoBold)
}
