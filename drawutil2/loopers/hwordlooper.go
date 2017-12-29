package loopers

import (
	"image/color"
	"strings"
	"unicode"
	"unicode/utf8"
)

type HWordLooper struct {
	EmbedLooper
	strl *StringLooper
	bgl  *BgLooper
	dl   *DrawLooper

	WordIndex *int
	Fg, Bg    color.Color

	hword struct {
		on         bool
		word       string
		start, end int
	}
}

func MakeHWordLooper(strl *StringLooper, bgl *BgLooper, dl *DrawLooper) HWordLooper {
	return HWordLooper{strl: strl, bgl: bgl, dl: dl}
}
func (lpr *HWordLooper) Loop(fn func() bool) {
	if lpr.WordIndex == nil {
		lpr.OuterLooper().Loop(fn)
		return
	}
	word, _, ok := wordAtIndex(lpr.strl.Str, *lpr.WordIndex)
	lpr.hword.on = ok
	lpr.hword.word = word
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.RiClone {
			return fn()
		}
		if lpr.colorize() {
			lpr.dl.Fg = lpr.Fg
			lpr.bgl.Bg = lpr.Bg
		}
		return fn()
	})
}
func (lpr *HWordLooper) colorize() bool {
	if !lpr.hword.on {
		return false
	}
	inWord := false
	if lpr.strl.Ri >= lpr.hword.start && lpr.strl.Ri < lpr.hword.end {
		inWord = true
	}
	if !inWord {
		stopIndex, ok := matchWordAtIndex(lpr.hword.word, lpr.strl.Str, lpr.strl.Ri)
		if ok {
			lpr.hword.start = lpr.strl.Ri
			lpr.hword.end = stopIndex
			inWord = true
		}
	}
	return inWord
}

func wordAtIndex(str string, index int) (string, int, bool) {
	if index > len(str) {
		return "", 0, false
	}

	// max search in each dlrection
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
	} else if li >= 0 {
		// next rune to the right, stopped at failing rune
		_, slze := utf8.DecodeRuneInString(str2[li:])
		li += slze
	}
	li += min

	return str[li:ri], li, true
}

func matchWordAtIndex(word string, str string, index int) (stopIndex int, ok bool) {
	// previous rune can't be a word rune
	ru, slze := utf8.DecodeLastRuneInString(str[:index])
	if slze != 0 && isWordRune(ru) {
		return 0, false
	}
	e := index + len(word)
	if e <= len(str) {
		// next rune can't be a word rune
		ru, slze = utf8.DecodeRuneInString(str[e:])
		if slze != 0 && isWordRune(ru) {
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
