package cmdutil

import (
	"image"
	"path/filepath"

	"github.com/jmigpin/editor/core/gosource"
	"github.com/jmigpin/editor/ui"
	"github.com/jmigpin/editor/util/drawutil/loopers"
)

func ToggleContextFloatBox(ed Editorer, p image.Point) {
	cfb := ed.UI().Root.ContextFloatBox
	cfb.Enabled = !cfb.Enabled
	UpdateContextFloatBox(ed, p)
}
func DisableContextFloatBox(ed Editorer) {
	cfb := ed.UI().Root.ContextFloatBox
	cfb.Enabled = false
	UpdateContextFloatBox(ed, image.Point{})
}
func UpdateContextFloatBox(ed Editorer, p image.Point) {
	cfb := ed.UI().Root.ContextFloatBox

	// ensure it's hidden if not enabled
	if !cfb.Enabled {
		cfb.ShowCalcMark(false)
		return
	}

	// find erow at p
	var erow ERower
	for _, e := range ed.ERowers() {
		if p.In(e.Row().Bounds) {
			erow = e
			break
		}
	}

	// no current erow, hide and leave
	if erow == nil {
		cfb.ShowCalcMark(false)
		return
	}

	// must be inside textarea (not toolbar)
	ta := erow.Row().TextArea
	if !p.In(ta.Bounds) {
		cfb.ShowCalcMark(false)
		return
	}

	// context defaults
	index, str := ta.CursorIndex(), ""
	var hopt *loopers.HighlightSegmentsOpt

	// context data
	switch filepath.Ext(erow.Filename()) {
	case ".go":
		ta := erow.Row().TextArea
		res, err := gosource.CodeCompletion(erow.Filename(), ta.Str(), ta.CursorIndex())
		if err != nil {
			//ed.Error(err)
			//log.Print(err)
			break
		}
		index = res.StartIndex
		str = res.Str
		if len(res.Segments) > 0 {
			fgbg := ui.UITheme.HighlightSegmentColors()
			hopt = &loopers.HighlightSegmentsOpt{
				Fg:              fgbg.Fg,
				Bg:              fgbg.Bg,
				OrderedSegments: res.Segments,
			}
		}
	}

	// set reference point
	p2 := ta.IndexPoint(index)
	p2.Y += ta.LineHeight()
	p2.X -= cfb.Label.Border.Left + cfb.Label.Pad.Left
	cfb.RefPoint = p2

	// set string and unhide
	cfb.Label.Text.Drawer.HighlightSegmentsOpt = hopt
	cfb.Label.Text.SetStr(str)
	cfb.ShowCalcMark(true)
}
