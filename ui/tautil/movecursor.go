package tautil

import "image"

func MoveCursorRight(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	_, i, ok := NextRuneIndex(ta.Str(), ta.CursorIndex())
	if !ok {
		return
	}
	ta.SetCursorIndex(i)
}
func MoveCursorLeft(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	_, i, ok := PreviousRuneIndex(ta.Str(), ta.CursorIndex())
	if !ok {
		return
	}
	ta.SetCursorIndex(i)
}

func MoveCursorToPoint(ta Texta, p *image.Point, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	index := ta.PointIndexFromOffset(p)
	ta.SetCursorIndex(index)
}

func MoveCursorUp(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	p := ta.IndexPoint266(ta.CursorIndex())
	p.Y -= ta.LineHeight()
	i := ta.Point266Index(p)
	ta.SetCursorIndex(i)
}
func MoveCursorDown(ta Texta, sel bool) {
	activateSelection(ta, sel)
	defer deactivateSelectionCheck(ta)
	p := ta.IndexPoint266(ta.CursorIndex())
	p.Y += ta.LineHeight()
	i := ta.Point266Index(p)
	ta.SetCursorIndex(i)
}
