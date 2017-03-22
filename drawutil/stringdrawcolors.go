package drawutil

import (
	"image"
	"image/color"
	"image/draw"
	"unicode"
	"unicode/utf8"
)

type StringDrawColors struct {
	sd     *StringDraw
	colors *Colors

	highlight bool       // set externally
	selection *Selection // set externally
}

type Colors struct {
	Fg, Bg      color.Color
	SelectionFg color.Color
	SelectionBg color.Color
	HighlightFg color.Color
	HighlightBg color.Color
	//Comment     color.Color
}

func DefaultColors() Colors {
	return Colors{
		Fg:          color.Black,
		Bg:          color.White,
		SelectionFg: color.Black,
		SelectionBg: color.Gray16{0x00ff},
		HighlightFg: color.Black,
		HighlightBg: color.Gray16{0x0fff},
		//Comment:     color.Gray16{0x000f},
	}
}

type Selection struct {
	StartIndex, EndIndex int
}

func NewStringDrawColors(img draw.Image, rect *image.Rectangle, face *Face, str string, colors *Colors) *StringDrawColors {
	sd := NewStringDraw(img, rect, face, str)
	return &StringDrawColors{sd: sd, colors: colors}
}
func (sdc *StringDrawColors) Loop() {
	liner := sdc.sd.liner

	// highlight cursor word
	highlightOn := false
	highlightWord := ""
	if sdc.highlight {
		highlightWord, highlightOn = wordAtIndex(liner.iter.str, sdc.sd.cursorIndex)
	}

	wordSt := false
	wordStStop := 0

	sdc.sd.Loop(func() (fg, bg color.Color, ok bool) {
		// rune color
		fg = sdc.colors.Fg

		//// comments
		//if liner.states.comment {
		//assignColorIfNotNil(&fg, sdc.colors.Comment)
		//}

		// highlight matching words
		if highlightOn {
			if !wordSt {
				stopIndex, ok := matchWordAtIndex(highlightWord, liner.iter.str, liner.iter.ri)
				if ok {
					wordSt = true
					wordStStop = stopIndex
				}
			}
			if wordSt {
				if liner.iter.ri == wordStStop {
					wordSt = false
				}
			}
			if wordSt {
				assignColorIfNotNil(&fg, sdc.colors.HighlightFg)
				assignColorIfNotNil(&bg, sdc.colors.HighlightBg)
			}
		}

		// selection
		if colorizeSelection(sdc.selection, liner) {
			assignColorIfNotNil(&fg, sdc.colors.SelectionFg)
			assignColorIfNotNil(&bg, sdc.colors.SelectionBg)
		}

		return fg, bg, true
	})
}

func assignColorIfNotNil(c *color.Color, c2 color.Color) {
	if c2 != nil {
		*c = c2
	}
}

func colorizeSelection(selection *Selection, liner *StringLiner) bool {
	if selection == nil {
		return false
	}
	if liner.isWrapLineRune {
		return false
	}
	s := selection.StartIndex
	e := selection.EndIndex
	ri := liner.iter.ri
	if s > e {
		s, e = e, s
	}
	return ri >= s && ri < e
}

func wordAtIndex(str string, index int) (string, bool) {
	if index > len(str) {
		return "", false
	}
	// back index
	i0 := index
	ru, size := utf8.DecodeRuneInString(str[i0:])
	if size == 0 || !isWordRune(ru) {
		return "", false
	}
	for {
		//println("currentword i0",i0,len(str))
		ru, size := utf8.DecodeLastRuneInString(str[:i0])
		if size == 0 {
			break
		}
		if !isWordRune(ru) {
			break
		}
		i0 -= size
	}
	// front index
	i1 := index
	for {
		ru, size := utf8.DecodeRuneInString(str[i1:])
		if size == 0 {
			break
		}
		if !isWordRune(ru) {
			break
		}
		i1 += size
	}
	if i0 == i1 {
		return "", false
	}
	return str[i0:i1], true
}
func isWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || ru == '_' || unicode.IsNumber(ru)
}
func matchWordAtIndex(word string, str string, index int) (int, bool) {
	w, ok := wordAtIndex(str, index)
	if !ok {
		return 0, false
	}
	if word == w {
		return index + len(word), true
	}
	return 0, false
}
