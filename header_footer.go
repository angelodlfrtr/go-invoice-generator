package generator

import (
	"fmt"

	"github.com/creasty/defaults"
	"github.com/jung-kurt/gofpdf"
)

// HeaderFooter define header or footer informations on document
type HeaderFooter struct {
	UseCustomFunc bool    `json:"-"`
	Text          string  `json:"text,omitempty"`
	FontSize      float64 `json:"font_size,omitempty" default:"7"`
	Pagination    bool    `json:"pagination,omitempty"`
}

type fnc func()

// ApplyFunc allow user to apply custom func
func (hf *HeaderFooter) ApplyFunc(pdf *gofpdf.Fpdf, fn fnc) {
	pdf.SetHeaderFunc(fn)
}

func (hf *HeaderFooter) applyHeader(doc *Document) error {
	if err := defaults.Set(hf); err != nil {
		return err
	}

	if !hf.UseCustomFunc {
		doc.pdf.SetHeaderFunc(func() {
			currentY := doc.pdf.GetY()
			currentX := doc.pdf.GetX()

			doc.pdf.SetTopMargin(HeaderMarginTop)
			doc.pdf.SetY(HeaderMarginTop)

			doc.pdf.SetLeftMargin(BaseMargin)
			doc.pdf.SetRightMargin(BaseMargin)

			// Parse Text as html (simple)
			doc.pdf.SetFont("Helvetica", "", hf.FontSize)
			_, lineHt := doc.pdf.GetFontSize()
			html := doc.pdf.HTMLBasicNew()
			html.Write(lineHt, hf.Text)

			// Apply pagination
			if !hf.Pagination {
				doc.pdf.AliasNbPages("") // Will replace {nb} with total page count
				doc.pdf.SetY(HeaderMarginTop + 8)
				doc.pdf.SetX(195)
				doc.pdf.CellFormat(10, 5, fmt.Sprintf("Page %d/{nb}", doc.pdf.PageNo()), "0", 0, "R", false, 0, "")
			}

			doc.pdf.SetY(currentY)
			doc.pdf.SetX(currentX)
			doc.pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
		})
	}

	return nil
}

func (hf *HeaderFooter) applyFooter(doc *Document) error {
	if err := defaults.Set(hf); err != nil {
		return err
	}

	if !hf.UseCustomFunc {
		doc.pdf.SetFooterFunc(func() {
			currentY := doc.pdf.GetY()
			currentX := doc.pdf.GetX()

			doc.pdf.SetTopMargin(HeaderMarginTop)
			doc.pdf.SetY(287 - HeaderMarginTop)

			// Parse Text as html (simple)
			doc.pdf.SetFont("Helvetica", "", hf.FontSize)
			_, lineHt := doc.pdf.GetFontSize()
			html := doc.pdf.HTMLBasicNew()
			html.Write(lineHt, hf.Text)

			// Apply pagination
			if hf.Pagination {
				doc.pdf.AliasNbPages("") // Will replace {nb} with total page count
				doc.pdf.SetY(287 - HeaderMarginTop - 8)
				doc.pdf.SetX(195)
				doc.pdf.CellFormat(10, 5, fmt.Sprintf("Page %d/{nb}", doc.pdf.PageNo()), "0", 0, "R", false, 0, "")
			}

			doc.pdf.SetY(currentY)
			doc.pdf.SetX(currentX)
			doc.pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
		})
	}

	return nil
}
