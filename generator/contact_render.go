package generator

import (
	"bytes"
	b64 "encoding/base64"
	"fmt"
	"image"

	"codeberg.org/go-pdf/fpdf"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// appendContact draws a party block (logo, name, address, identifiers and
// additional lines) at x, y and returns the Y position below it.
func (r *renderer) appendContact(c *invoice.Contact, x, y float64) float64 {
	r.pdf.SetXY(x, y)

	if c.Logo != nil {
		fileName := b64.StdEncoding.EncodeToString([]byte(c.Name))
		_, format, _ := image.DecodeConfig(bytes.NewReader(c.Logo))

		imageInfo := r.pdf.RegisterImageOptionsReader(fileName, fpdf.ImageOptions{
			ImageType: format,
		}, bytes.NewReader(c.Logo))

		if imageInfo != nil {
			var imageOpt fpdf.ImageOptions
			imageOpt.ImageType = format
			r.pdf.ImageOptions(fileName, r.pdf.GetX(), y, 0, 30, false, imageOpt, 0, "")
			r.pdf.SetY(y + 30)
		}
	}

	r.setFillColor(r.opts.GreyBgColor)
	r.pdf.SetX(x)
	r.pdf.Rect(x, r.pdf.GetY(), 70, 8, "F")
	r.pdf.SetFont(r.opts.BoldFont, "B", 10)
	r.pdf.Cell(40, 8, r.encode(c.Name))
	r.pdf.SetFont(r.opts.Font, "", 10)

	if c.Address != nil {
		var addrRectHeight float64 = 17
		if len(c.Address.Address2) > 0 {
			addrRectHeight += 5
		}
		if len(c.Address.Country) == 0 {
			addrRectHeight -= 5
		}
		r.pdf.Rect(x, r.pdf.GetY()+9, 70, addrRectHeight, "F")
		r.pdf.SetFont(r.opts.Font, "", 10)
		r.pdf.SetXY(x, r.pdf.GetY()+10)
		r.pdf.MultiCell(70, 5, r.encode(c.Address.ToString()), "0", "L", false)
	}

	// Identifiers, then the caller's additional lines, in small text.
	var lines []string
	if c.VATID != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", r.opts.TextContactVATID, c.VATID))
	}
	if c.LegalID != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", r.opts.TextContactLegalID, c.LegalID))
	}
	lines = append(lines, c.AdditionalInfo...)

	if len(lines) > 0 {
		r.pdf.SetFontSize(SmallTextFontSize)
		r.pdf.SetXY(x, r.pdf.GetY()+2)
		for _, line := range lines {
			r.pdf.SetXY(x, r.pdf.GetY())
			r.pdf.MultiCell(70, 3, r.encode(line), "0", "L", false)
		}
		r.pdf.SetXY(x, r.pdf.GetY())
		r.pdf.SetFontSize(BaseTextFontSize)
	}

	return r.pdf.GetY()
}
