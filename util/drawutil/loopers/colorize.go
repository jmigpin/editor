package loopers

import (
	"image/color"
	"strings"
	"unicode"
)

// Colorize comments and strings.
type Colorize struct {
	EmbedLooper
	opt  *ColorizeOpt
	strl *String
	data ColorizeData
}

func MakeColorize(strl *String, opt *ColorizeOpt) Colorize {
	return Colorize{strl: strl, opt: opt}
}
func (lpr *Colorize) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		if lpr.strl.RiClone {
			return fn()
		}

		switch lpr.data.state {
		case CDStateNormal:
			if lpr.opt.Comment.Line != "" {
				prefix := lpr.opt.Comment.Line
				if strings.HasPrefix(lpr.strl.Str[lpr.strl.Ri:], prefix) {
					lpr.data.state = CDStateCommentLine
				}
			}
			if lpr.opt.Comment.Enclosed[0] != "" {
				prefix := lpr.opt.Comment.Enclosed[0]
				if strings.HasPrefix(lpr.strl.Str[lpr.strl.Ri:], prefix) {
					lpr.data.state = CDStateCommentEnclosed
					lpr.data.index = lpr.strl.Ri + len(prefix)
				}
			}
			if lpr.strl.Ru == '`' || unicode.In(lpr.strl.Ru, unicode.Quotation_Mark) {
				lpr.data.state = CDStateString
				lpr.data.quote = lpr.strl.Ru
				lpr.data.index = lpr.strl.Ri + len(string(lpr.strl.Ru))
			}
		case CDStateCommentEnclosedClosing:
			if lpr.strl.Ri >= lpr.data.index {
				lpr.data.state = CDStateNormal
			}
		case CDStateString:
			// TODO: colorize strings should be optional (if any) for .txt files

			// escape rune inside string
			if lpr.strl.Ru == '\\' {
				lpr.data.state = CDStateStringEscape
				lpr.data.index = lpr.strl.Ri + len(string(lpr.strl.Ru))
			}
			// End string state for non-multiline quote if end of line. Allows .txt files to have squote.
			if lpr.strl.Ru == '\n' && lpr.data.quote != '`' {
				lpr.data.state = CDStateNormal
			}
		}

		r := fn()

		switch lpr.data.state {
		case CDStateCommentLine:
			if lpr.strl.Ru == '\n' {
				lpr.data.state = CDStateNormal
			}
		case CDStateCommentEnclosed:
			if lpr.strl.Ri >= lpr.data.index {
				prefix := lpr.opt.Comment.Enclosed[1]
				if strings.HasPrefix(lpr.strl.Str[lpr.strl.Ri:], prefix) {
					lpr.data.state = CDStateCommentEnclosedClosing
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
func (lpr *Colorize) KeepPosData() interface{} {
	return lpr.data
}

// Implements PosDataKeeper
func (lpr *Colorize) RestorePosData(data interface{}) {
	lpr.data = data.(ColorizeData)
}

type ColorizeData struct {
	state CDState
	quote rune
	index int
}

// colorize data state
type CDState int

const (
	CDStateNormal CDState = iota
	CDStateCommentLine
	CDStateCommentEnclosed
	CDStateCommentEnclosedClosing
	CDStateString
	CDStateStringEscape
)

type ColorizeOpt struct {
	Comment ColorizeCommentOpt
}

type ColorizeCommentOpt struct {
	Line     string    // ex: "//", "#"
	Enclosed [2]string // ex: "/*" "*/"
	Fg       color.Color
}

type ColorizeColor struct {
	EmbedLooper
	colorize *Colorize
	dl       *Draw
}

func MakeColorizeColor(dl *Draw, colorize *Colorize) ColorizeColor {
	return ColorizeColor{dl: dl, colorize: colorize}
}
func (lpr *ColorizeColor) Loop(fn func() bool) {
	lpr.OuterLooper().Loop(func() bool {
		switch lpr.colorize.data.state {
		case CDStateCommentLine, CDStateCommentEnclosed, CDStateCommentEnclosedClosing:
			lpr.dl.Fg = lpr.colorize.opt.Comment.Fg
		case CDStateString, CDStateStringEscape:
			//lpr.dl.Fg = lpr.colorize.opt.Comment.Fg
		}
		return fn()
	})
}
