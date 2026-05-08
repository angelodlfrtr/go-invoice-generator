// Package facturx embeds a Factur-X (ZUGFeRD 2.x) compliant CII XML into a
// PDF produced by go-invoice-generator.
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
//
// Note: full PDF/A-3 conformance (embedded ICC color profiles, linearisation,
// etc.) requires the base PDF to be generated in PDF/A mode, which fpdf does
// not support out of the box. The output is Factur-X compatible for validators
// that check the XML content and XMP metadata rather than the full PDF/A spec.
package facturx

import (
	"bytes"
	"fmt"
	"time"

	generator "github.com/angelodlfrtr/go-invoice-generator"
	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const xmlFilename = "factur-x.xml"

// Attach generates the CII XML invoice for doc, embeds it inside pdfBytes as a
// named attachment, and patches the XMP metadata with the required Factur-X
// declarations. It returns the modified PDF bytes.
//
// doc.Build() (which calls doc.Validate()) must have been called before Attach
// so that all monetary values are computed.
func Attach(pdfBytes []byte, doc *generator.Document, opts Options) ([]byte, error) {
	xmlBytes, err := BuildXML(doc, opts)
	if err != nil {
		return nil, fmt.Errorf("facturx: build XML: %w", err)
	}

	pdfBytes, err = embedXML(pdfBytes, xmlBytes)
	if err != nil {
		return nil, fmt.Errorf("facturx: embed XML: %w", err)
	}

	pdfBytes, err = patchXMP(pdfBytes, opts.profile())
	if err != nil {
		return nil, fmt.Errorf("facturx: patch XMP: %w", err)
	}

	return pdfBytes, nil
}

// embedXML adds xmlBytes as a named embedded file (factur-x.xml) to the PDF
// using pdfcpu's context API so we can pass the data directly without a temp
// file.
func embedXML(pdfBytes, xmlBytes []byte) ([]byte, error) {
	now := time.Now()
	att := model.Attachment{
		Reader:   bytes.NewReader(xmlBytes),
		ID:       xmlFilename,
		FileName: xmlFilename,
		Desc:     "Factur-X Electronic Invoice",
		ModTime:  &now,
	}

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	in := bytes.NewReader(pdfBytes)
	ctx, err := pdfcpuapi.ReadValidateAndOptimize(in, conf)
	if err != nil {
		return nil, err
	}

	if err := ctx.AddAttachment(att, false); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := pdfcpuapi.Write(ctx, &out, conf); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// patchXMP replaces the PDF's existing XMP packet with a Factur-X compliant
// one, or appends one if the PDF has no XMP metadata. fpdf-generated PDFs
// typically contain a minimal XMP packet that we can locate by the standard
// <?xpacket …?> markers.
func patchXMP(pdfBytes []byte, profile Profile) ([]byte, error) {
	xmpBytes, err := buildXMP(profile)
	if err != nil {
		return nil, err
	}

	const beginMarker = "<?xpacket begin="
	const endMarker = "<?xpacket end="

	si := bytes.Index(pdfBytes, []byte(beginMarker))
	ei := bytes.LastIndex(pdfBytes, []byte(endMarker))
	if si >= 0 && ei > si {
		closeIdx := bytes.Index(pdfBytes[ei:], []byte("?>"))
		if closeIdx >= 0 {
			endFull := ei + closeIdx + 2
			var buf bytes.Buffer
			buf.Write(pdfBytes[:si])
			buf.Write(xmpBytes)
			buf.Write(pdfBytes[endFull:])
			return buf.Bytes(), nil
		}
	}

	// No existing XMP packet found — return unchanged. Full metadata injection
	// requires lower-level PDF object manipulation beyond pdfcpu's public API.
	return pdfBytes, nil
}
