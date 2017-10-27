package tautil

import (
	"image"
	"strings"
)

func MoveCursorToPoint(ta Texta, p *image.Point, sel bool) {
	p2 := p.Sub(ta.Bounds().Min)
	p2.Y += ta.OffsetY()
	i := ta.GetIndex(&p2)
	updateSelection(ta, sel, i)

	// set primary copy
	if ta.SelectionOn() {
		a, b := SelectionStringIndexes(ta)
		s := ta.Str()[a:b]
		ta.SetPrimaryCopy(s)
	}
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
	p := ta.GetPoint(ta.CursorIndex())
	p.Y -= ta.LineHeight() - 1
	i := ta.GetIndex(&p)
	updateSelection(ta, sel, i)
}
func MoveCursorDown(ta Texta, sel bool) {
	p := ta.GetPoint(ta.CursorIndex())
	p.Y += ta.LineHeight() + 1
	i := ta.GetIndex(&p)
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
	i := strings.LastIndexFunc(str[:index], endOfNextWord())
	if i < 0 {
		i = 0
	} else {
		i++
	}
	return i
}
func jumpRightIndex(str string, index int) int {
	i := strings.IndexFunc(str[index:], endOfNextWord())
	if i < 0 {
		i = len(str[index:])
	}
	return index + i
}

func endOfNextWord() func(rune) bool {
	first := true
	var inWord bool
	return func(ru rune) bool {
		w := isWordRune(ru)
		if first {
			first = false
			inWord = w
		} else {
			if !inWord {
				inWord = w
			} else {
				if !w {
					return true
				}
			}
		}
		return false
	}
}
