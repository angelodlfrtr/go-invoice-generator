package generator

import (
	"fmt"

	"github.com/creasty/defaults"
	"github.com/jung-kurt/gofpdf"
)

// HeaderFooter define header or footer informations on document
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

	if !hf.UseCustomFunc {
		pdf.SetHeaderFunc(func() {
			currentY := pdf.GetY()
			currentX := pdf.GetX()

			pdf.SetTopMargin(HeaderMarginTop)
			pdf.SetY(HeaderMarginTop)

			pdf.SetLeftMargin(BaseMargin)
			pdf.SetRightMargin(BaseMargin)

			// Parse Text as html (simple)
			pdf.SetFont("Helvetica", "", hf.FontSize)
			_, lineHt := pdf.GetFontSize()
			html := pdf.HTMLBasicNew()
			html.Write(lineHt, hf.Text)

			// Apply pagination
			if !hf.Pagination {
				pdf.AliasNbPages("") // Will replace {nb} with total page count
				pdf.SetY(HeaderMarginTop + 8)
				pdf.SetX(195)
				pdf.CellFormat(10, 5, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "0", 0, "R", false, 0, "")
			}

			pdf.SetY(currentY)
			pdf.SetX(currentX)
			pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
		})
	}

	return nil
}

func (hf *HeaderFooter) applyFooter(d *Document, pdf *gofpdf.Fpdf) error {
	if err := defaults.Set(hf); err != nil {
		return err
	}

	if !hf.UseCustomFunc {
		pdf.SetFooterFunc(func() {
			currentY := pdf.GetY()
			currentX := pdf.GetX()

			pdf.SetTopMargin(HeaderMarginTop)
			pdf.SetY(287 - HeaderMarginTop)

			// Parse Text as html (simple)
			pdf.SetFont("Helvetica", "", hf.FontSize)
			_, lineHt := pdf.GetFontSize()
			html := pdf.HTMLBasicNew()
			html.Write(lineHt, hf.Text)

			// Apply pagination
			if hf.Pagination {
				pdf.AliasNbPages("") // Will replace {nb} with total page count
				pdf.SetY(287 - HeaderMarginTop - 8)
				pdf.SetX(195)
				pdf.CellFormat(10, 5, fmt.Sprintf("Page %d/{nb}", pdf.PageNo()), "0", 0, "R", false, 0, "")
			}

			pdf.SetY(currentY)
			pdf.SetX(currentX)
			pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
		})
	}

	return nil
}
