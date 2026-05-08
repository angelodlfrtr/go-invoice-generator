// Package facturx embeds a Factur-X (ZUGFeRD 2.x) compliant CII XML into a
// PDF produced by go-invoice-generator and brings the document into PDF/A-3b
// conformance by adding an sRGB OutputIntent and the required XMP declarations.
//
// Typical usage:
//
//	pdf, _ := doc.Build()
//	var buf bytes.Buffer
//	pdf.Output(&buf)
//
//	result, err := facturx.Attach(buf.Bytes(), doc, facturx.Options{
//	    Profile:     facturx.ProfileMinimum,
//	    SellerTaxID: "FR12345678901",
//	})
package facturx

import (
	"bytes"
	"fmt"

	generator "github.com/angelodlfrtr/go-invoice-generator"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const xmlFilename = "factur-x.xml"

// Attach generates the CII XML invoice for doc, embeds it inside pdfBytes as a
// named attachment with /AFRelationship /Alternative, adds a PDF/A-3b OutputIntent
// with an embedded sRGB ICC profile, and merges the required Factur-X XMP
// declarations into the PDF's existing XMP packet.
//
// doc.Build() (which calls doc.Validate()) must have been called before Attach
// so that all monetary values are computed.
func Attach(pdfBytes []byte, doc *generator.Document, opts Options) ([]byte, error) {
	xmlBytes, err := BuildXML(doc, opts)
	if err != nil {
		return nil, fmt.Errorf("facturx: build XML: %w", err)
	}

	result, err := attachFX(pdfBytes, xmlBytes, opts)
	if err != nil {
		return nil, fmt.Errorf("facturx: %w", err)
	}

	return result, nil
}

// attachFX is the single-pass PDF manipulation pipeline: load into a pdfcpu
// context, embed the XML with proper PDF/A-3 semantics, write once.
func attachFX(pdfBytes, xmlBytes []byte, opts Options) ([]byte, error) {
	profile := opts.profile()

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	ctx, err := pdfcpuapi.ReadValidateAndOptimize(bytes.NewReader(pdfBytes), conf)
	if err != nil {
		return nil, fmt.Errorf("read PDF: %w", err)
	}

	if err := embedXMLWithAFRelationship(ctx, xmlBytes); err != nil {
		return nil, fmt.Errorf("embed XML: %w", err)
	}

	if err := addOutputIntent(ctx.XRefTable); err != nil {
		return nil, fmt.Errorf("output intent: %w", err)
	}

	if err := ensureXMPInContext(ctx, profile); err != nil {
		return nil, fmt.Errorf("ensure XMP: %w", err)
	}

	if opts.ShowIcon {
		if err := stampFXIcon(ctx, profile); err != nil {
			return nil, fmt.Errorf("stamp icon: %w", err)
		}
	}

	var out bytes.Buffer
	if err := pdfcpuapi.Write(ctx, &out, conf); err != nil {
		return nil, fmt.Errorf("write PDF: %w", err)
	}

	return out.Bytes(), nil
}
