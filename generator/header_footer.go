package generator

import (
	"fmt"

	"codeberg.org/go-pdf/fpdf"
	"github.com/creasty/defaults"
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
func (hf *HeaderFooter) ApplyFunc(pdf *fpdf.Fpdf, fn fnc) {
	pdf.SetHeaderFunc(fn)
}

// applyHeader apply header to document
func (hf *HeaderFooter) applyHeader(doc *renderer) error {
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
			doc.pdf.SetFont(doc.opts.Font, "", hf.FontSize)
			_, lineHt := doc.pdf.GetFontSize()
			html := doc.pdf.HTMLBasicNew()
			html.Write(lineHt, doc.encode(hf.Text))

			// Apply pagination
			if !hf.Pagination {
				doc.pdf.AliasNbPages("") // Will replace {nb} with total page count
				doc.pdf.SetY(HeaderMarginTop + 8)
				doc.pdf.SetX(195)
				doc.pdf.CellFormat(
					10,
					5,
					doc.encode(fmt.Sprintf("Page %d/{nb}", doc.pdf.PageNo())),
					"0",
					0,
					"R",
					false,
					0,
					"",
				)
			}

			doc.pdf.SetY(currentY)
			doc.pdf.SetX(currentX)
			doc.pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
		})
	}

	return nil
}

// applyFooter apply footer to document
func (hf *HeaderFooter) applyFooter(doc *renderer) error {
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
			doc.pdf.SetFont(doc.opts.Font, "", hf.FontSize)
			_, lineHt := doc.pdf.GetFontSize()
			html := doc.pdf.HTMLBasicNew()
			html.Write(lineHt, doc.encode(hf.Text))

			// Apply pagination
			if hf.Pagination {
				doc.pdf.AliasNbPages("") // Will replace {nb} with total page count
				doc.pdf.SetY(287 - HeaderMarginTop - 8)
				doc.pdf.SetX(195)
				doc.pdf.CellFormat(
					10,
					5,
					doc.encode(fmt.Sprintf("Page %d/{nb}", doc.pdf.PageNo())),
					"0",
					0,
					"R",
					false,
					0,
					"",
				)
			}

			doc.pdf.SetY(currentY)
			doc.pdf.SetX(currentX)
			doc.pdf.SetMargins(BaseMargin, BaseMarginTop, BaseMargin)
		})
	}

	return nil
}
