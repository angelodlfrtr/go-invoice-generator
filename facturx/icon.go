package facturx

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

//go:embed fxicons/factur_x_minimum.png
var iconMinimum []byte

//go:embed fxicons/factur_x_basic_wl.png
var iconBasicWL []byte

//go:embed fxicons/factur_x_basic.png
var iconBasic []byte

//go:embed fxicons/factur-x_en_16931.png
var iconEN16931 []byte

//go:embed fxicons/factur-x_extended.png
var iconExtended []byte

func iconForProfile(p Profile) []byte {
	switch p {
	case ProfileBasicWL:
		return iconBasicWL
	case ProfileBasic:
		return iconBasic
	case ProfileEN16931:
		return iconEN16931
	case ProfileExtended:
		return iconExtended
	default:
		return iconMinimum
	}
}

// stampFXIcon places the profile icon in the bottom-right corner of page 1.
// It manually creates the image XObject to maintain PDF/A-3B conformance:
// no OCG, no Interpolate:true, no ExtGState transparency.
func stampFXIcon(ctx *model.Context, profile Profile) error {
	xrt := ctx.XRefTable
	icon := iconForProfile(profile)

	// Create image XObject. CreateImageStreamDict sets Interpolate:true for
	// small images; override it to false before inserting (clause 6.2.8.3).
	sd, w, h, err := model.CreateImageStreamDict(xrt, bytes.NewReader(icon))
	if err != nil {
		return fmt.Errorf("create image stream: %w", err)
	}
	sd.Update("Interpolate", types.Boolean(false))
	imgIR, err := xrt.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("image indref: %w", err)
	}

	// Locate page 1 and its resource dict.
	pageDict, _, inhAttrs, err := xrt.PageDict(1, false)
	if err != nil {
		return fmt.Errorf("page dict: %w", err)
	}

	// Add image to page XObject resources.
	const imgID = "FxIcon"
	if err := addIconToPageResources(xrt, pageDict, imgID, *imgIR); err != nil {
		return fmt.Errorf("add resource: %w", err)
	}

	// Determine page dimensions (points).
	mb := inhAttrs.MediaBox
	if obj, found := pageDict.Find("MediaBox"); found {
		if arr, ok := obj.(types.Array); ok && len(arr) == 4 {
			mb = types.RectForArray(arr)
		}
	}
	pgW := 595.276 // A4 fallback
	if mb != nil {
		pgW = mb.Width()
	}

	// Target: 5 % of page width, placed at bottom-right with a small margin.
	imgW := pgW * 0.05
	imgH := imgW * float64(h) / float64(w)
	margin := pgW * 0.02
	x := pgW - imgW - margin
	y := margin

	// Append PDF content stream operators that draw the image.
	ops := fmt.Sprintf(
		"q %.4f 0 0 %.4f %.4f %.4f cm /%s Do Q\n",
		imgW, imgH, x, y, imgID,
	)
	if err := xrt.AppendContent(pageDict, []byte(ops)); err != nil {
		return fmt.Errorf("append content: %w", err)
	}

	return nil
}

// addIconToPageResources merges imgID → imgIR into the page's /XObject resource dict.
func addIconToPageResources(xrt *model.XRefTable, pageDict types.Dict, imgID string, imgIR types.IndirectRef) error {
	resObj, hasRes := pageDict.Find("Resources")

	var resDict types.Dict
	if hasRes {
		switch v := resObj.(type) {
		case types.Dict:
			resDict = v
		case types.IndirectRef:
			entry, found := xrt.FindTableEntryForIndRef(&v)
			if !found || entry == nil {
				resDict = types.NewDict()
			} else if d, ok := entry.Object.(types.Dict); ok {
				resDict = d
			} else {
				resDict = types.NewDict()
			}
		default:
			resDict = types.NewDict()
		}
	} else {
		resDict = types.NewDict()
		pageDict.Insert("Resources", resDict)
	}

	xoObj, hasXO := resDict.Find("XObject")
	if !hasXO {
		resDict.Insert("XObject", types.Dict(map[string]types.Object{imgID: imgIR}))
		return nil
	}
	switch v := xoObj.(type) {
	case types.Dict:
		v.Insert(imgID, imgIR)
	case types.IndirectRef:
		entry, found := xrt.FindTableEntryForIndRef(&v)
		if found && entry != nil {
			if d, ok := entry.Object.(types.Dict); ok {
				d.Insert(imgID, imgIR)
			}
		}
	}
	return nil
}
