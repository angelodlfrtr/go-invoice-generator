package facturx

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// fxXMPBlocksRaw are the three rdf:Description blocks injected into the
// existing XMP packet. They must be placed before </rdf:RDF>.
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

// mergeXMPInContext locates the PDF's existing XMP metadata stream via the pdfcpu
// context, decodes it, injects Factur-X rdf:Description blocks before </rdf:RDF>,
// and re-encodes the stream. All changes stay within the context so that pdfcpu
// writes correct byte offsets when serialising the PDF.
func mergeXMPInContext(ctx *model.Context, profile Profile) error {
	fxBlocks, err := buildFXXMPBlocks(profile)
	if err != nil {
		return err
	}

	catDict, err := ctx.Catalog()
	if err != nil {
		return fmt.Errorf("catalog: %w", err)
	}

	metaRef, ok := catDict["Metadata"].(types.IndirectRef)
	if !ok {
		return nil // no XMP packet — skip silently
	}

	entry, found := ctx.FindTableEntryForIndRef(&metaRef)
	if !found || entry == nil {
		return nil
	}

	sd, ok := entry.Object.(types.StreamDict)
	if !ok {
		return nil
	}

	if err := sd.Decode(); err != nil {
		return fmt.Errorf("decode XMP stream: %w", err)
	}

	const rdfClose = "</rdf:RDF>"
	idx := bytes.LastIndex(sd.Content, []byte(rdfClose))
	if idx < 0 {
		return nil // malformed XMP — skip
	}

	var merged bytes.Buffer
	merged.Write(sd.Content[:idx])
	merged.Write(fxBlocks)
	merged.Write(sd.Content[idx:])
	sd.Content = merged.Bytes()

	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode XMP stream: %w", err)
	}

	entry.Object = sd
	return nil
}
