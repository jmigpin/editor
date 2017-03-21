package tautil

import (
	"image"

	"github.com/jmigpin/editor/drawutil"
)

func MoveCursorRight(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	_, i, ok := NextRuneIndex(ta.Str(), ta.CursorIndex())
	if !ok {
		return
	}
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(ta.CursorIndex())
}
func MoveCursorLeft(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	_, i, ok := PreviousRuneIndex(ta.Str(), ta.CursorIndex())
	if !ok {
		return
	}
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(ta.CursorIndex())
}

func MoveCursorToPoint(ta Texta, p *image.Point, sel bool) {
	updateSelectionState(ta, sel)
	p2 := p.Sub(ta.Bounds().Min)
	p3 := drawutil.PointToPoint266(&p2)
	p3.Y += ta.OffsetY()
	index := ta.PointIndex(p3)
	ta.SetCursorIndex(index)
}

func MoveCursorUp(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	p := ta.IndexPoint(ta.CursorIndex())
	p.Y -= ta.LineHeight()
	i := ta.PointIndex(p)
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(ta.CursorIndex())

	//// adjust offset if it stops being visible
	//y1 := (p.Y - ta.OffsetY()).Round()
	//if y1 < 0 {
	//// push offset one line up
	//ta.SetOffsetY(ta.OffsetY() - ta.LineHeight())
	//}
}
func MoveCursorDown(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	p := ta.IndexPoint(ta.CursorIndex())
	p.Y += ta.LineHeight()
	i := ta.PointIndex(p)
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(ta.CursorIndex())

	//// adjust offset if it stops being visible
	//// need to get point again to have the cursor with limits checked
	//p = ta.IndexPoint(ta.CursorIndex())
	//p.Y += ta.LineHeight()
	//y1 := (p.Y - ta.OffsetY()).Round()
	//if y1 > ta.Bounds().Dy() {
	//// push offset one line down
	//ta.SetOffsetY(ta.OffsetY() + ta.LineHeight())
	//}
}

//func visibleCursorPos() {
//p := ta.IndexPoint(ta.CursorIndex())
//p.Y -= ta.OffsetY()

//b := ta.Bounds()
//p2 := drawutil.Point266ToPoint(p).Add(b.Min)
//y2 := p2.Y + LineHeight().Round()

//if p2.Y < b.Min.Y {
//ta.OffsetY(fixed.I(b.Min.Y))
//}else if p2.Y > b.Max.Y{

//}
