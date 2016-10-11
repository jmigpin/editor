package tautil

import "image"

func MoveCursorRight(ta Texta, sel bool) {
	activateSelection(ta, sel)
	_, i, ok := NextRuneIndex(ta.Text(), ta.CursorIndex())
	if !ok {
		return
	}
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}
func MoveCursorLeft(ta Texta, sel bool) {
	activateSelection(ta, sel)
	_, i, ok := PreviousRuneIndex(ta.Text(), ta.CursorIndex())
	if !ok {
		return
	}
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}

func MoveCursorToPoint(ta Texta, p *image.Point, sel bool) {
	activateSelection(ta, sel)
	index := ta.PointIndexFromOffset(p)
	ta.SetCursorIndex(index)
	deactivateSelectionCheck(ta)
}

func MoveCursorUp(ta Texta, sel bool) {
	activateSelection(ta, sel)
	p := ta.IndexPoint266(ta.CursorIndex())
	p.Y -= ta.LineHeight()
	i := ta.Point266Index(p)
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}
func MoveCursorDown(ta Texta, sel bool) {
	activateSelection(ta, sel)
	p := ta.IndexPoint266(ta.CursorIndex())
	p.Y += ta.LineHeight()
	i := ta.Point266Index(p)
	ta.SetCursorIndex(i)
	deactivateSelectionCheck(ta)
}
