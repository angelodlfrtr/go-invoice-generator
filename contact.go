package generator

import (
	"bytes"
	b64 "encoding/base64"
	"image"

	"github.com/jung-kurt/gofpdf"
)

// Contact contact a company informations
type Contact struct {
	Name    string   `json:"name,omitempty" validate:"required,min=1,max=256"`
	Logo    *[]byte  `json:"logo,omitempty"` // Logo byte array
	Address *Address `json:"address,omitempty"`
}

func (c *Contact) appendContactTODoc(
	x float64,
	y float64,
	fill bool,
	logoAlign string,
	doc *Document,
) float64 {
	doc.pdf.SetXY(x, y)

	// Logo
	if c.Logo != nil {
		// Create filename
		fileName := b64.StdEncoding.EncodeToString([]byte(c.Name))
		// Create reader from logo bytes
		ioReader := bytes.NewReader(*c.Logo)
		// Get image format
		_, format, _ := image.DecodeConfig(bytes.NewReader(*c.Logo))
		// Register image in pdf
		imageInfo := doc.pdf.RegisterImageOptionsReader(fileName, gofpdf.ImageOptions{
			ImageType: format,
		}, ioReader)

		if imageInfo != nil {
			var imageOpt gofpdf.ImageOptions
			imageOpt.ImageType = format

			doc.pdf.ImageOptions(fileName, doc.pdf.GetX(), y, 0, 30, false, imageOpt, 0, "")

			doc.pdf.SetY(y + 30)
		}
	}

	// Name
	if fill {
		doc.pdf.SetFillColor(
			doc.Options.GreyBgColor[0],
			doc.Options.GreyBgColor[1],
			doc.Options.GreyBgColor[2],
		)
	} else {
		doc.pdf.SetFillColor(255, 255, 255)
	}

	// Reset x
	doc.pdf.SetX(x)

	// Name rect
	doc.pdf.Rect(x, doc.pdf.GetY(), 70, 8, "F")

	// Set name
	doc.pdf.SetFont("Helvetica", "B", 10)
	doc.pdf.Cell(40, 8, c.Name)
	doc.pdf.SetFont("Helvetica", "", 10)

	if c.Address != nil {
		// Address rect
		var addrRectHeight float64 = 17

		if len(c.Address.Address2) > 0 {
			addrRectHeight = addrRectHeight + 5
		}

		if len(c.Address.Country) == 0 {
			addrRectHeight = addrRectHeight - 5
		}

		doc.pdf.Rect(x, doc.pdf.GetY()+9, 70, addrRectHeight, "F")

		// Set address
		doc.pdf.SetFont("Helvetica", "", 10)
		doc.pdf.SetXY(x, doc.pdf.GetY()+10)
		doc.pdf.MultiCell(70, 5, c.Address.ToString(), "0", "L", false)
	}

	return doc.pdf.GetY()
}

func (c *Contact) appendCompanyContactToDoc(doc *Document) float64 {
	x, y, _, _ := doc.pdf.GetMargins()
	return c.appendContactTODoc(x, y, true, "L", doc)
}

func (c *Contact) appendCustomerContactToDoc(doc *Document) float64 {
	return c.appendContactTODoc(130, BaseMarginTop+25, true, "R", doc)
}
