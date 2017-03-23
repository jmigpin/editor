package tautil

import (
	"image"
	"strings"

	"github.com/jmigpin/editor/drawutil"
)

func MoveCursorToPoint(ta Texta, p *image.Point, sel bool) {
	updateSelectionState(ta, sel)

	p2 := p.Sub(ta.Bounds().Min)
	p3 := drawutil.PointToPoint266(&p2)
	p3.Y += ta.OffsetY()
	index := ta.PointIndex(p3)

	ta.SetCursorIndex(index)
}

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

func MoveCursorUp(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	p := ta.IndexPoint(ta.CursorIndex())
	p.Y -= ta.LineHeight()
	i := ta.PointIndex(p)
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(ta.CursorIndex())
}
func MoveCursorDown(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	p := ta.IndexPoint(ta.CursorIndex())
	p.Y += ta.LineHeight()
	i := ta.PointIndex(p)
	ta.SetCursorIndex(i)
	ta.MakeIndexVisible(ta.CursorIndex())
}

func MoveCursorJumpLeft(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	i := jumpLeftIndex(ta.Str(), ta.CursorIndex())
	ta.SetCursorIndex(i)
}
func MoveCursorJumpRight(ta Texta, sel bool) {
	updateSelectionState(ta, sel)
	i := jumpRightIndex(ta.Str(), ta.CursorIndex())
	ta.SetCursorIndex(i)
}

func jumpLeftIndex(str string, index int) int {
	typ := 0
	fn := func(ru rune) bool {
		typ2 := jumpType(ru)
		if typ == 0 {
			typ = typ2
			return false
		}
		return typ2 != typ
	}
	i := strings.LastIndexFunc(str[:index], fn)
	if i < 0 {
		i = 0
	} else {
		i++
	}
	return i
}
func jumpRightIndex(str string, index int) int {
	typ := 0
	fn := func(ru rune) bool {
		typ2 := jumpType(ru)
		if typ == 0 {
			typ = typ2
			return false
		}
		return typ2 != typ
	}
	i := strings.IndexFunc(str[index:], fn)
	if i < 0 {
		i = len(str[index:])
	}
	return index + i
}
func jumpType(ru rune) int {
	if isWordRune(ru) {
		return 1
	}
	return 2
}
