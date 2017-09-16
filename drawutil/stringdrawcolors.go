package drawutil

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font"
)

type StringDrawColors struct {
	sd     *StringDraw
	colors *Colors

	highlight bool       // set externally
	selection *Selection // set externally
}

func NewStringDrawColors(img draw.Image, rect *image.Rectangle, face font.Face, str string, colors *Colors) *StringDrawColors {
	sd := NewStringDraw(img, rect, face, str)
	return &StringDrawColors{sd: sd, colors: colors}
}
func (sdc *StringDrawColors) Loop() {
	liner := sdc.sd.liner

	// highlight cursor word
	hOn := false
	hWord := ""
	if sdc.highlight {
		hWord, _, hOn = wordAtIndex(liner.iter.str, sdc.sd.cursorIndex)
	}

	wordSt := false
	wordStStop := 0

	sdc.sd.Loop(func() (fg, bg color.Color, ok bool) {
		// highlight matching words
		if hOn {
			if !wordSt {
				stopIndex, ok := matchWordAtIndex(hWord, liner.iter.str, liner.iter.ri)
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
			if wordSt && !liner.isWrapLineRune {
				fg = sdc.colors.HighlightFg
				bg = sdc.colors.HighlightBg
			}
		}

		// selection
		if colorizeSelection(sdc.selection, liner) {
			fg = sdc.colors.SelectionFg
			bg = sdc.colors.SelectionBg
		}

		// default rune color
		if fg == nil {
			fg = sdc.colors.Fg
		}
		return fg, bg, true
	})
}

type Colors struct {
	Fg, Bg      color.Color
	SelectionFg color.Color
	SelectionBg color.Color
	HighlightFg color.Color
	HighlightBg color.Color
}

func DefaultColors() Colors {
	return Colors{
		Fg:          color.Black,
		Bg:          color.White,
		SelectionFg: color.Black,
		SelectionBg: color.Gray16{0x00ff},
		HighlightFg: color.Black,
		HighlightBg: color.Gray16{0x0fff},
	}
}

type Selection struct {
	StartIndex, EndIndex int
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

func wordAtIndex(str string, index int) (string, int, bool) {
	if index > len(str) {
		return "", 0, false
	}

	// max search in each direction
	cap := 50

	// right limit
	max := index + cap
	if max > len(str) {
		max = len(str)
	}
	str2 := str[index:max]
	str3 := str2 + " " // allow to find on eos
	i := strings.IndexFunc(str3, isNotWordRune)
	if i <= 0 {
		// either not found until eos (cap), or first rune failed
		return "", 0, false
	}
	ri := index + i

	// left limit
	min := index - cap
	if min < 0 {
		min = 0
	}
	str2 = str[min:index]
	li := strings.LastIndexFunc(str2, isNotWordRune)
	if li < 0 {
		li = 0
	} else if li > 0 {
		// next rune to the right, stopped at failing rune
		_, size := utf8.DecodeRuneInString(str2[li:])
		li += size
	}
	li += min

	return str[li:ri], li, true
}
func matchWordAtIndex(word string, str string, index int) (int, bool) {
	// previous rune can't be a word rune
	ru, size := utf8.DecodeLastRuneInString(str[:index])
	if size != 0 && isWordRune(ru) {
		return 0, false
	}
	e := index + len(word)
	if e <= len(str) {
		// next rune can't be a word rune
		ru, size = utf8.DecodeRuneInString(str[e:])
		if size != 0 && isWordRune(ru) {
			return 0, false
		}
		// match words
		if word == str[index:e] {
			return e, true
		}
	}
	return 0, false
}
func isNotWordRune(ru rune) bool {
	return !isWordRune(ru)
}
func isWordRune(ru rune) bool {
	return unicode.IsLetter(ru) || ru == '_' || unicode.IsDigit(ru)
}
