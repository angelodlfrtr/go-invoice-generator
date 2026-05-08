package facturx

import (
	"bytes"
	_ "embed"
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

//go:embed srgb.icc
var srgbICCProfile []byte

// embedXMLWithAFRelationship attaches xmlBytes as factur-x.xml inside the PDF context,
// sets /AFRelationship /Alternative on the file spec, and registers the file spec
// in the catalog's /AF array.
func embedXMLWithAFRelationship(ctx *model.Context, xmlBytes []byte) error {
	xrt := ctx.XRefTable
	if err := xrt.LocateNameTree("EmbeddedFiles", true); err != nil {
		return fmt.Errorf("locate name tree: %w", err)
	}

	streamIR, err := xrt.NewEmbeddedStreamDict(bytes.NewReader(xmlBytes), time.Now())
	if err != nil {
		return fmt.Errorf("embedded stream: %w", err)
	}
	// PDF/A-3 clause 6.8: the EmbeddedFile stream must carry a MIME Subtype.
	// pdfcpu encodes '/' as '#2f' when writing Name tokens.
	if entry, found := xrt.FindTableEntryForIndRef(streamIR); found && entry != nil {
		if sd, ok := entry.Object.(types.StreamDict); ok {
			sd.InsertName("Subtype", "text/xml")
			entry.Object = sd
		}
	}

	fsDict, err := xrt.NewFileSpecDict(xmlFilename, xmlFilename, "Factur-X Electronic Invoice", *streamIR)
	if err != nil {
		return fmt.Errorf("file spec dict: %w", err)
	}
	fsDict.InsertName("AFRelationship", "Alternative")

	fsIR, err := xrt.IndRefForNewObject(fsDict)
	if err != nil {
		return fmt.Errorf("filespec indref: %w", err)
	}

	m := model.NameMap{xmlFilename: []types.Dict{fsDict}}
	if err := xrt.Names["EmbeddedFiles"].Add(xrt, xmlFilename, *fsIR, m, []string{"F", "UF"}); err != nil {
		return fmt.Errorf("name tree add: %w", err)
	}

	catDict, err := xrt.Catalog()
	if err != nil {
		return fmt.Errorf("catalog: %w", err)
	}
	catDict["AF"] = types.Array{*fsIR}

	return nil
}

// addOutputIntent inserts a PDF/A-3 OutputIntent with an embedded sRGB ICC profile
// into the catalog, satisfying the /DestOutputProfile requirement.
func addOutputIntent(xrt *model.XRefTable) error {
	sd, err := xrt.NewStreamDictForBuf(srgbICCProfile)
	if err != nil {
		return fmt.Errorf("icc stream dict: %w", err)
	}
	sd.InsertInt("N", 3)
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode icc stream: %w", err)
	}
	iccIR, err := xrt.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("icc indref: %w", err)
	}

	oiDict := types.NewDict()
	oiDict.InsertName("Type", "OutputIntent")
	oiDict.InsertName("S", "GTS_PDFA1")
	oiDict.InsertString("OutputConditionIdentifier", "sRGB IEC61966-2.1")
	oiDict.InsertString("Info", "sRGB IEC61966-2.1")
	oiDict.Insert("DestOutputProfile", *iccIR)

	oiIR, err := xrt.IndRefForNewObject(oiDict)
	if err != nil {
		return fmt.Errorf("output intent indref: %w", err)
	}

	catDict, err := xrt.Catalog()
	if err != nil {
		return fmt.Errorf("catalog: %w", err)
	}
	catDict["OutputIntents"] = types.Array{*oiIR}

	return nil
}
