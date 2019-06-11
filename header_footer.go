package generator

import (
	"fmt"
	"github.com/creasty/defaults"
	"github.com/jung-kurt/gofpdf"
)

type HeaderFooter struct {
	UseCustomFunc bool
	Text          string
	FontSize      float64 `default:"7"`
	Pagination    bool
}

type fnc func()

// ApplyFunc allow user to apply custom func
func (hf *HeaderFooter) ApplyFunc(pdf *gofpdf.Fpdf, fn fnc) {
	pdf.SetHeaderFunc(fn)
}

func (hf *HeaderFooter) applyHeader(d *Document, pdf *gofpdf.Fpdf) error {
	if err := defaults.Set(hf); err != nil {
		return err
	}

	if hf.UseCustomFunc == false {
		pdf.SetHeaderFunc(func() {
			currentY := pdf.GetY()
			currentX := pdf.GetX()

			pdf.SetTopMargin(HEADER_MARGIN_TOP)
			pdf.SetY(HEADER_MARGIN_TOP)

			pdf.SetLeftMargin(BASE_MARGIN)
			pdf.SetRightMargin(BASE_MARGIN)

			// Parse Text as html (simple)
			pdf.SetFont("Helvetica", "", hf.FontSize)
			_, lineHt := pdf.GetFontSize()
			html := pdf.HTMLBasicNew()
			html.Write(lineHt, hf.Text)

			// Apply pagination
			if hf.Pagination == true {
				pdf.AliasNbPages("") // Will replace {nb} with total page count
				pdf.SetY(HEADER_MARGIN_TOP + 8)
				pdf.SetX(195)
				pdf.CellFormat(10, 5, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "0", 0, "R", false, 0, "")
			}

			pdf.SetY(currentY)
			pdf.SetX(currentX)
			pdf.SetMargins(BASE_MARGIN, BASE_MARGIN_TOP, BASE_MARGIN)
		})
	}

	return nil
}

func (hf *HeaderFooter) applyFooter(d *Document, pdf *gofpdf.Fpdf) error {
	if err := defaults.Set(hf); err != nil {
		return err
	}

	if hf.UseCustomFunc == false {
		pdf.SetFooterFunc(func() {
			currentY := pdf.GetY()
			currentX := pdf.GetX()

			pdf.SetTopMargin(HEADER_MARGIN_TOP)
			pdf.SetY(287 - HEADER_MARGIN_TOP)

			// Parse Text as html (simple)
			pdf.SetFont("Helvetica", "", hf.FontSize)
			_, lineHt := pdf.GetFontSize()
			html := pdf.HTMLBasicNew()
			html.Write(lineHt, hf.Text)

			// Apply pagination
			if hf.Pagination == true {
				pdf.AliasNbPages("") // Will replace {nb} with total page count
				pdf.SetY(287 - HEADER_MARGIN_TOP - 8)
				pdf.SetX(195)
				pdf.CellFormat(10, 5, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "0", 0, "R", false, 0, "")
			}

			pdf.SetY(currentY)
			pdf.SetX(currentX)
			pdf.SetMargins(BASE_MARGIN, BASE_MARGIN_TOP, BASE_MARGIN)
		})
	}

	return nil
}
