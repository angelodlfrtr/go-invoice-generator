package facturx

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// fxXMPBlocksRaw are the three rdf:Description blocks injected into the XMP
// packet. They must appear before </rdf:RDF>.
var fxXMPBlocksRaw = `
    <rdf:Description rdf:about=""
        xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/">
      <pdfaid:part>3</pdfaid:part>
      <pdfaid:conformance>B</pdfaid:conformance>
    </rdf:Description>

    <rdf:Description rdf:about=""
        xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"
        xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#"
        xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#">
      <pdfaExtension:schemas>
        <rdf:Bag>
          <rdf:li rdf:parseType="Resource">
            <pdfaSchema:schema>Factur-X PDFA Extension Schema</pdfaSchema:schema>
            <pdfaSchema:namespaceURI>urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#</pdfaSchema:namespaceURI>
            <pdfaSchema:prefix>fx</pdfaSchema:prefix>
            <pdfaSchema:property>
              <rdf:Seq>
                <rdf:li rdf:parseType="Resource">
                  <pdfaProperty:name>DocumentFileName</pdfaProperty:name>
                  <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                  <pdfaProperty:category>external</pdfaProperty:category>
                  <pdfaProperty:description>The name of the embedded XML invoice file</pdfaProperty:description>
                </rdf:li>
                <rdf:li rdf:parseType="Resource">
                  <pdfaProperty:name>DocumentType</pdfaProperty:name>
                  <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                  <pdfaProperty:category>external</pdfaProperty:category>
                  <pdfaProperty:description>The type of the hybrid document in uppercase letters</pdfaProperty:description>
                </rdf:li>
                <rdf:li rdf:parseType="Resource">
                  <pdfaProperty:name>Version</pdfaProperty:name>
                  <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                  <pdfaProperty:category>external</pdfaProperty:category>
                  <pdfaProperty:description>The version of the standard applying to the embedded XML invoice file</pdfaProperty:description>
                </rdf:li>
                <rdf:li rdf:parseType="Resource">
                  <pdfaProperty:name>ConformanceLevel</pdfaProperty:name>
                  <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                  <pdfaProperty:category>external</pdfaProperty:category>
                  <pdfaProperty:description>The conformance level of the embedded XML invoice file</pdfaProperty:description>
                </rdf:li>
              </rdf:Seq>
            </pdfaSchema:property>
          </rdf:li>
        </rdf:Bag>
      </pdfaExtension:schemas>
    </rdf:Description>

    <rdf:Description rdf:about=""
        xmlns:fx="urn:factur-x:pdfa:CrossIndustryDocument:invoice:1p0#">
      <fx:DocumentFileName>factur-x.xml</fx:DocumentFileName>
      <fx:DocumentType>INVOICE</fx:DocumentType>
      <fx:Version>1.0</fx:Version>
      <fx:ConformanceLevel>{{.Profile}}</fx:ConformanceLevel>
    </rdf:Description>
`

type xmpData struct {
	Profile string
}

// buildFXXMPBlocks returns the rendered Factur-X rdf:Description blocks.
func buildFXXMPBlocks(profile Profile) ([]byte, error) {
	tmpl, err := template.New("xmp").Parse(fxXMPBlocksRaw)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, xmpData{Profile: profile.String()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildFullXMP wraps fxBlocks in a minimal well-formed XMP packet.
// The BOM (U+FEFF) in the xpacket begin attribute is injected via string
// concatenation because a literal BOM in a Go source file causes a compile error.
func buildFullXMP(fxBlocks []byte) []byte {
	var b bytes.Buffer
	b.WriteString("<?xpacket begin=\"\xEF\xBB\xBF\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n")
	b.WriteString("<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n")
	b.WriteString("  <rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n")
	b.Write(fxBlocks)
	b.WriteString("  </rdf:RDF>\n")
	b.WriteString("</x:xmpmeta>\n")
	b.WriteString("<?xpacket end=\"w\"?>")
	return b.Bytes()
}

// ensureXMPInContext ensures the PDF catalog has a /Metadata stream that carries
// the required Factur-X XMP declarations. If no metadata stream exists (fpdf does
// not generate one), a new uncompressed stream is created and wired into the
// catalog. If one already exists, the Factur-X blocks are merged in before
// </rdf:RDF>.
//
// PDF/A-3 requires the metadata stream to be uncompressed (clause 6.6.2.1) and
// to carry /Type /Metadata and /Subtype /XML.
func ensureXMPInContext(ctx *model.Context, profile Profile) error {
	fxBlocks, err := buildFXXMPBlocks(profile)
	if err != nil {
		return err
	}

	catDict, err := ctx.Catalog()
	if err != nil {
		return fmt.Errorf("catalog: %w", err)
	}

	metaRef, hasExisting := catDict["Metadata"].(types.IndirectRef)
	if hasExisting {
		entry, found := ctx.FindTableEntryForIndRef(&metaRef)
		if found && entry != nil {
			if sd, ok := entry.Object.(types.StreamDict); ok {
				if err := sd.Decode(); err == nil {
					const rdfClose = "</rdf:RDF>"
					if idx := bytes.LastIndex(sd.Content, []byte(rdfClose)); idx >= 0 {
						var merged bytes.Buffer
						merged.Write(sd.Content[:idx])
						merged.Write(fxBlocks)
						merged.Write(sd.Content[idx:])
						sd.Content = merged.Bytes()
					}
				}
				setMetadataStreamMeta(&sd)
				if err := sd.Encode(); err != nil {
					return fmt.Errorf("encode XMP stream: %w", err)
				}
				entry.Object = sd
				return nil
			}
		}
	}

	// No metadata stream present: create one from scratch.
	sd := types.StreamDict{
		Dict:    types.NewDict(),
		Content: buildFullXMP(fxBlocks),
	}
	setMetadataStreamMeta(&sd)
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode new XMP stream: %w", err)
	}
	ir, err := ctx.IndRefForNewObject(sd)
	if err != nil {
		return fmt.Errorf("xmp indref: %w", err)
	}
	catDict["Metadata"] = *ir
	return nil
}

// setMetadataStreamMeta stamps the stream dict with the mandatory PDF/A-3
// metadata entries and removes any compression filter (clause 6.6.2.1 forbids
// filtering the document metadata stream).
func setMetadataStreamMeta(sd *types.StreamDict) {
	sd.InsertName("Type", "Metadata")
	sd.InsertName("Subtype", "XML")
	sd.FilterPipeline = nil
	sd.Dict.Delete("Filter")
	sd.Dict.Delete("DecodeParms")
}
