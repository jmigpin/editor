package tautil

import (
	"image"
	"strings"

	"golang.org/x/image/math/fixed"
)

func MoveCursorToPoint(ta Texta, p *image.Point, sel bool) {
	p2 := p.Sub(ta.Bounds().Min)
	p3 := fixed.P(p2.X, p2.Y)
	p3.Y += ta.OffsetY()
	i := ta.PointIndex(&p3)
	updateSelection(ta, sel, i)
}

func MoveCursorRight(ta Texta, sel bool) {
	_, i, ok := NextRuneIndex(ta.Str(), ta.CursorIndex())
	if !ok {
		return
	}
	updateSelection(ta, sel, i)
}
func MoveCursorLeft(ta Texta, sel bool) {
	_, i, ok := PreviousRuneIndex(ta.Str(), ta.CursorIndex())
	if !ok {
		return
	}
	updateSelection(ta, sel, i)
}

func MoveCursorUp(ta Texta, sel bool) {
	p := ta.IndexPoint(ta.CursorIndex())
	p.Y -= ta.LineHeight()
	i := ta.PointIndex(p)
	updateSelection(ta, sel, i)
}
func MoveCursorDown(ta Texta, sel bool) {
	p := ta.IndexPoint(ta.CursorIndex())
	p.Y += ta.LineHeight()
	i := ta.PointIndex(p)
	updateSelection(ta, sel, i)
}

func MoveCursorJumpLeft(ta Texta, sel bool) {
	i := jumpLeftIndex(ta.Str(), ta.CursorIndex())
	updateSelection(ta, sel, i)
}
func MoveCursorJumpRight(ta Texta, sel bool) {
	i := jumpRightIndex(ta.Str(), ta.CursorIndex())
	updateSelection(ta, sel, i)
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
