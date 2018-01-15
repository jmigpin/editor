package loopers

import (
	"image/color"
	"strings"
	"unicode"
)

type Comments struct {
	EmbedLooper
	opt  *CommentsOpt
	strl *String
	data CommentsData
}

func MakeComments(strl *String, opt *CommentsOpt) Comments {
	return Comments{strl: strl, opt: opt}
}
func (lpr *Comments) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.RiClone {
			return fn()
		}

		switch lpr.data.state {
		case CDStateNormal:
			if lpr.opt.Line != "" {
				prefix := lpr.opt.Line
				if strings.HasPrefix(lpr.strl.Str[lpr.strl.Ri:], prefix) {
					lpr.data.state = CDStateLine
				}
			}
			if lpr.opt.Enclosed[0] != "" {
				prefix := lpr.opt.Enclosed[0]
				if strings.HasPrefix(lpr.strl.Str[lpr.strl.Ri:], prefix) {
					lpr.data.state = CDStateEnclosed
					lpr.data.index = lpr.strl.Ri + len(prefix)
				}
			}
			if lpr.strl.Ru == '`' || unicode.In(lpr.strl.Ru, unicode.Quotation_Mark) {
				lpr.data.state = CDStateString
				lpr.data.quote = lpr.strl.Ru
				lpr.data.index = lpr.strl.Ri + len(string(lpr.strl.Ru))
			}
		case CDStateEnclosedClosing:
			if lpr.strl.Ri >= lpr.data.index {
				lpr.data.state = CDStateNormal
			}
		case CDStateString:
			if lpr.strl.Ru == '\\' {
				lpr.data.state = CDStateStringEscape
				lpr.data.index = lpr.strl.Ri + len(string(lpr.strl.Ru))
			}
		}

		r := fn()

		switch lpr.data.state {
		case CDStateLine:
			if lpr.strl.Ru == '\n' {
				lpr.data.state = CDStateNormal
			}
		case CDStateEnclosed:
			if lpr.strl.Ri >= lpr.data.index {
				prefix := lpr.opt.Enclosed[1]
				if strings.HasPrefix(lpr.strl.Str[lpr.strl.Ri:], prefix) {
					lpr.data.state = CDStateEnclosedClosing
					lpr.data.index = lpr.strl.Ri + len(prefix)
				}
			}
		case CDStateString:
			if lpr.strl.Ri >= lpr.data.index {
				if lpr.strl.Ru == lpr.data.quote {
					lpr.data.state = CDStateNormal
				}
			}
		case CDStateStringEscape:
			if lpr.strl.Ri >= lpr.data.index {
				lpr.data.state = CDStateString
			}
		}

		return r
	})
}

// Implements PosDataKeeper
func (lpr *Comments) KeepPosData() interface{} {
	return lpr.data
}

// Implements PosDataKeeper
func (lpr *Comments) RestorePosData(data interface{}) {
	lpr.data = data.(CommentsData)
}

type CDState int

const (
	CDStateNormal CDState = iota
	CDStateLine
	CDStateEnclosed
	CDStateEnclosedClosing
	CDStateString
	CDStateStringEscape
)

type CommentsData struct {
	state CDState
	quote rune
	index int
}

type CommentsOpt struct {
	Line     string    // ex: "//", "#"
	Enclosed [2]string // ex: "/*" "*/"
	Fg       color.Color
}

type CommentsColor struct {
	EmbedLooper
	comments *Comments
	dl       *Draw
}

func MakeCommentsColor(dl *Draw, comments *Comments) CommentsColor {
	return CommentsColor{dl: dl, comments: comments}
}
func (lpr *CommentsColor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.comments.data.state {
		case CDStateLine, CDStateEnclosed, CDStateEnclosedClosing:
			lpr.dl.Fg = lpr.comments.opt.Fg
		}
		return fn()
	})
}
