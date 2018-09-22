package drawer3

import (
	"image/draw"

	"github.com/jmigpin/editor/util/imageutil"
	"github.com/jmigpin/editor/util/mathutil"
)

type BgFill struct {
	EExt
	cc *CurColors

	// setup values
	img draw.Image
}

func BgFill1(cc *CurColors) BgFill {
	return BgFill{cc: cc}
}

func (bgf *BgFill) setup(img draw.Image) {
	bgf.img = img
}

func (bgf *BgFill) Iterate(r *ExtRunner) {
	if bgf.cc.Bg != nil {
		offset := mathutil.PIntf2(r.D.Offset())
		pos := r.D.Bounds().Min
		pb := r.RR.OffsetPenBoundsRect(offset, pos)
		pb = pb.Intersect(r.D.Bounds())
		imageutil.FillRectangle(bgf.img, &pb, bgf.cc.Bg)
	}
	r.NextExt()
}
