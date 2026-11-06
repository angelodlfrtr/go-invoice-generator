package facturx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ErrNoXML is returned by Extract when the PDF carries no e-invoice XML.
var ErrNoXML = errors.New("no Factur-X / ZUGFeRD XML attachment found")

// knownXMLNames are the attachment names used by Factur-X, ZUGFeRD 2 and
// XRechnung hybrids, in order of preference.
var knownXMLNames = []string{"factur-x.xml", "zugferd-invoice.xml", "xrechnung.xml", "ZUGFeRD-invoice.xml"}

// Extract returns the e-invoice XML embedded in a Factur-X / ZUGFeRD PDF.
// It looks for the standard attachment names first and falls back to the
// first XML attachment.
func Extract(pdfBytes []byte) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	attachments, err := pdfcpuapi.ExtractAttachmentsRaw(bytes.NewReader(pdfBytes), "", nil, conf)
	if err != nil {
		if strings.Contains(err.Error(), "no attachments") {
			return nil, fmt.Errorf("facturx: %w", ErrNoXML)
		}
		return nil, fmt.Errorf("facturx: read attachments: %w", err)
	}

	var pick *model.Attachment
	for _, name := range knownXMLNames {
		for i := range attachments {
			if strings.EqualFold(attachments[i].FileName, name) {
				pick = &attachments[i]
				break
			}
		}
		if pick != nil {
			break
		}
	}
	if pick == nil {
		for i := range attachments {
			if strings.HasSuffix(strings.ToLower(attachments[i].FileName), ".xml") {
				pick = &attachments[i]
				break
			}
		}
	}
	if pick == nil {
		return nil, fmt.Errorf("facturx: %w", ErrNoXML)
	}

	data, err := io.ReadAll(pick.Reader)
	if err != nil {
		return nil, fmt.Errorf("facturx: read %s: %w", pick.FileName, err)
	}
	return data, nil
}

// Read extracts and parses the e-invoice embedded in a Factur-X PDF.
func Read(pdfBytes []byte) (*Parsed, error) {
	xmlBytes, err := Extract(pdfBytes)
	if err != nil {
		return nil, err
	}
	return Parse(xmlBytes)
}
