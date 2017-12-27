package cmdutil

import (
	"image"
	"path/filepath"

	"github.com/jmigpin/editor/core/gosource"
)

func ToggleContextFloatBox(ed Editorer, p image.Point) {
	cfb := ed.UI().Layout.ContextFloatBox
	cfb.Enabled = !cfb.Enabled
	UpdateContextFloatBox(ed, p)
}
func UpdateContextFloatBox(ed Editorer, p image.Point) {
	cfb := ed.UI().Layout.ContextFloatBox

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

	// get context
	index, str := ta.CursorIndex(), ""
	switch filepath.Ext(erow.Filename()) {
	case ".go":
		index2, str2, err := codeCompletion(erow)
		if err != nil {
			//log.Print(err)
		} else {
			index = index2
			str = str2
		}
	}

	// set reference point
	p2 := ta.IndexPoint(index)
	p2.Y += ta.LineHeight()
	p2.X -= cfb.Label.Border.Left + cfb.Label.Pad.Left
	cfb.RefPoint = p2

	// set string and unhide
	cfb.SetStr("**TODO/TESTING**\n" + str)
	cfb.ShowCalcMark(true)
}

func codeCompletion(erow ERower) (int, string, error) {
	ta := erow.Row().TextArea
	index2, objs, err := gosource.CodeCompletion(erow.Filename(), ta.Str(), ta.CursorIndex())
	if err != nil {
		return 0, "", err
	}
	str := gosource.FormatObjs(objs)
	return index2, str, nil
}
