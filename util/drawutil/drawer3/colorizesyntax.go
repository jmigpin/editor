package drawer3

import (
	"image/color"
	"unicode"

	"github.com/jmigpin/editor/util/iout"
)

type ColorizeSyntax struct {
	EExt
	Opt ColorizeSyntaxOpt
	d   Drawer // needed by SetOn

	// setup values
	data        ColorizeSyntaxData
	normalFuncs []func(r *ExtRunner)
	cline       []rune
	cenc0       []rune
}

func ColorizeSyntax1(d Drawer) ColorizeSyntax {
	return ColorizeSyntax{d: d}
}

func (cs *ColorizeSyntax) SetOn(v bool) {
	if v != cs.EExt.On() {
		cs.d.SetNeedMeasure(true)
	}
	cs.EExt.SetOn(v)
}

func (cs *ColorizeSyntax) Start(r *ExtRunner) {
	cs.data = ColorizeSyntaxData{}

	cs.normalFuncs = nil
	if cs.Opt.Comment.Line != "" {
		cs.normalFuncs = append(cs.normalFuncs, cs.commentLine)
		cs.cline = []rune(cs.Opt.Comment.Line)
	}
	if cs.Opt.Comment.Enclosed[0] != "" {
		cs.normalFuncs = append(cs.normalFuncs, cs.commentEnc0)
		cs.cenc0 = []rune(cs.Opt.Comment.Enclosed[0])
	}
	cs.normalFuncs = append(cs.normalFuncs, cs.quotes)
}

func (cs *ColorizeSyntax) Iterate(r *ExtRunner) {
	if r.RR.RiClone() {
		r.NextExt()
		return
	}

	cs.preState(r)

	if !r.NextExt() {
		return
	}

	cs.postState(r)
}

//----------

func (cs *ColorizeSyntax) preState(r *ExtRunner) {
	switch cs.data.state {
	case CSSNormal:
		for _, f := range cs.normalFuncs {
			f(r)
		}
	case CSSCommentEnclosed:
		if r.RR.Ri >= cs.data.index {
			prefix := []byte(cs.Opt.Comment.Enclosed[1])
			if iout.HasPrefix(r.RR.reader, r.RR.Ri, prefix) {
				cs.data.state = CSSCommentEnclosedClosing
				cs.data.index = r.RR.Ri + len(prefix)
			}
		}
	case CSSCommentEnclosedClosing:
		if r.RR.Ri >= cs.data.index {
			cs.data.state = CSSNormal
		}
	case CSSString:
		// escape rune inside string
		if r.RR.Ru == '\\' {
			cs.data.state = CSSStringEscape
			cs.data.index = r.RR.Ri + len(string(r.RR.Ru))
		}
		// End string state for non-multiline quote if end of line. Allows .txt files to have squote (ex: "don't").
		if r.RR.Ru == '\n' && cs.data.quote != '`' {
			cs.data.state = CSSNormal
		}
	}
}

func (cs *ColorizeSyntax) commentLine(r *ExtRunner) {
	if r.RR.Ru == cs.cline[0] {
		s := []byte(string(cs.cline))
		if iout.HasPrefix(r.RR.reader, r.RR.Ri, s) {
			cs.data.state = CSSCommentLine
		}
	}
}
func (cs *ColorizeSyntax) commentEnc0(r *ExtRunner) {
	if r.RR.Ru == cs.cenc0[0] {
		s := []byte(string(cs.cenc0))
		if iout.HasPrefix(r.RR.reader, r.RR.Ri, s) {
			cs.data.state = CSSCommentEnclosed
			cs.data.index = r.RR.Ri + len(s)
		}
	}
}
func (cs *ColorizeSyntax) quotes(r *ExtRunner) {
	if r.RR.Ru == '`' || unicode.In(r.RR.Ru, unicode.Quotation_Mark) {
		cs.data.state = CSSString
		cs.data.quote = r.RR.Ru
		cs.data.index = r.RR.Ri + len(string(r.RR.Ru))
	}
}

//----------

func (cs *ColorizeSyntax) postState(r *ExtRunner) {
	switch cs.data.state {
	case CSSCommentLine:
		if r.RR.Ru == '\n' {
			cs.data.state = CSSNormal
		}

	case CSSString:
		if r.RR.Ri >= cs.data.index {
			if r.RR.Ru == cs.data.quote {
				cs.data.state = CSSNormal
			}
		}
	case CSSStringEscape:
		if r.RR.Ri >= cs.data.index {
			cs.data.state = CSSString
		}
	}
}

//----------

// Implements PosDataKeeper
func (cs *ColorizeSyntax) KeepPosData() interface{} {
	return cs.data
}

// Implements PosDataKeeper
func (cs *ColorizeSyntax) RestorePosData(data interface{}) {
	cs.data = data.(ColorizeSyntaxData)
}

//----------

type ColorizeSyntaxData struct {
	state CSState
	quote rune
	index int
}

//----------

// colorize syntax state
type CSState int

const (
	CSSNormal CSState = iota
	CSSCommentLine
	CSSCommentEnclosed
	CSSCommentEnclosedClosing
	CSSString
	CSSStringEscape
)

//----------

type ColorizeSyntaxOpt struct {
	String struct {
		Fg color.Color
	}
	Comment struct {
		Line     string    // ex: "//", "#"
		Enclosed [2]string // ex: "/*" "*/"
		Fg       color.Color
	}
}

//----------

type ColorizeSyntaxColor struct {
	EExt
	csyntax *ColorizeSyntax
	cc      *CurColors
}

func ColorizeSyntaxColor1(csyntax *ColorizeSyntax, cc *CurColors) ColorizeSyntaxColor {
	return ColorizeSyntaxColor{csyntax: csyntax, cc: cc}
}

func (csc *ColorizeSyntaxColor) Iterate(r *ExtRunner) {
	if !csc.csyntax.On() {
		r.NextExt()
		return
	}

	switch csc.csyntax.data.state {
	case CSSString, CSSStringEscape:
		fg := csc.csyntax.Opt.String.Fg
		if fg != nil {
			csc.cc.Fg = fg
		}
	case CSSCommentLine, CSSCommentEnclosed, CSSCommentEnclosedClosing:
		fg := csc.csyntax.Opt.Comment.Fg
		if fg != nil {
			csc.cc.Fg = fg
		}
	}
	r.NextExt()
}
