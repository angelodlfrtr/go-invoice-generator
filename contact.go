package generator

import (
	"bufio"
	"bytes"
	b64 "encoding/base64"
	"image"
	"os"

	"github.com/jung-kurt/gofpdf"
)

// Contact contact a company informations
type Contact struct {
	Name    string  `validate:"required,min=1,max=256"`
	Logo    *[]byte // Logo byte array
	Address *Address
}

func (c *Contact) appendContactTODoc(x float64, y float64, fill bool, logoAlign string, pdf *gofpdf.Fpdf) float64 {
	pdf.SetXY(x, y)

	// Logo
	if c.Logo != nil {
		// Create filename
		fileName := b64.StdEncoding.EncodeToString([]byte(c.Name))
		// Create reader from logo bytes
		ioReader := bytes.NewReader(*c.Logo)
		// Get image format
		_, format, _ := image.DecodeConfig(bytes.NewReader(*c.Logo))
		// Register image in pdf
		imageInfo := pdf.RegisterImageOptionsReader(fileName, gofpdf.ImageOptions{
			ImageType: format,
		}, ioReader)

		if imageInfo != nil {
			var imageOpt gofpdf.ImageOptions
			imageOpt.ImageType = format

			pdf.ImageOptions(fileName, pdf.GetX(), y, 0, 30, false, imageOpt, 0, "")

			pdf.SetY(y + 30)
		}
	}

	// Name
	if fill {
		pdf.SetFillColor(GreyBgColor[0], GreyBgColor[1], GreyBgColor[2])
	} else {
		pdf.SetFillColor(255, 255, 255)
	}

	// Reset x
	pdf.SetX(x)

	// Name rect
	pdf.Rect(x, pdf.GetY(), 70, 8, "F")

	// Set name
	pdf.SetFont("Helvetica", "B", 10)
	pdf.Cell(40, 8, c.Name)
	pdf.SetFont("Helvetica", "", 10)

	if c.Address != nil {
		// Address rect
		var addrRectHeight float64 = 17

		if len(c.Address.Address2) > 0 {
			addrRectHeight = addrRectHeight + 5
		}

		if len(c.Address.Country) == 0 {
			addrRectHeight = addrRectHeight - 5
		}

		pdf.Rect(x, pdf.GetY()+9, 70, addrRectHeight, "F")

		// Set address
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(x, pdf.GetY()+10)
		pdf.MultiCell(70, 5, c.Address.ToString(), "0", "L", false)
	}

	return pdf.GetY()
}

func (c *Contact) appendCompanyContactToDoc(pdf *gofpdf.Fpdf) float64 {
	x, y, _, _ := pdf.GetMargins()
	return c.appendContactTODoc(x, y, true, "L", pdf)
}

func (c *Contact) appendCustomerContactToDoc(pdf *gofpdf.Fpdf) float64 {
	return c.appendContactTODoc(130, BaseMarginTop+25, true, "R", pdf)
}

// ImageToBytes converts a picture into a byte[] when the
// relative path is passed as an parameter.
func ImageToBytes(logo string) ([]byte, error) {

	// Relative Path
	file, err := os.Open(logo)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	logoBytes := make([]byte, size)

	// read file into bytes
	buffer := bufio.NewReader(file)
	_, err = buffer.Read(logoBytes)

	if err != nil {
		return nil, err
	}

	return logoBytes, nil
}
