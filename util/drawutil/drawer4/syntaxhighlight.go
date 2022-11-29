package drawer4

import (
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil"
)

func updateSyntaxHighlightOps(d *Drawer) {
	if !d.Opt.SyntaxHighlight.On {
		d.Opt.SyntaxHighlight.Group.Ops = nil
		return
	}

	if d.opt.syntaxH.updated {
		return
	}
	d.opt.syntaxH.updated = true

	pad := 2500
	sh := &SyntaxHighlight{d: d}
	d.Opt.SyntaxHighlight.Group.Ops = sh.do(pad)
}

//----------

type SyntaxHighlight struct {
	d   *Drawer
	sc  *parseutil.Scanner
	ops []*ColorizeOp
}

func (sh *SyntaxHighlight) do(pad int) []*ColorizeOp {
	// limit reading to be able to handle big content
	o, n, _, _ := sh.d.visibleLen()
	min, max := o, o+n

	r := iorw.NewLimitedReaderAtPad(sh.d.reader, min, max, pad)

	sh.sc = parseutil.NewScanner()
	sh.sc.SetSrc2(r)

	for !sh.sc.M.Eof() {
		sh.normal(pad)
	}
	return sh.ops
}
func (sh *SyntaxHighlight) normal(pad int) {
	pos0 := sh.sc.KeepPos()
	opt := &sh.d.Opt.SyntaxHighlight
	switch {
	case sh.comments():
		// ok
	case sh.sc.M.StringSection("\"", '\\', true, pad, false) == nil ||
		sh.sc.M.StringSection("'", '\\', true, 4, false) == nil:

		// unable to support multiline quotes (Ex: Go backquotes) since the whole file is not parsed, just a section.
		// Also, in the case of Go backquotes, probably only .go files should support them.

		op1 := &ColorizeOp{
			Offset: pos0.Pos,
			Fg:     opt.String.Fg,
			Bg:     opt.String.Bg,
		}
		op2 := &ColorizeOp{Offset: sh.sc.Pos}
		sh.ops = append(sh.ops, op1, op2)
	default:
		_, _ = sh.sc.ReadRune()
	}
}

func (sh *SyntaxHighlight) comments() bool {
	opt := &sh.d.Opt.SyntaxHighlight
	for _, c := range opt.Comment.Defs {
		if sh.comment(c) {
			return true
		}
	}
	return false
}
func (sh *SyntaxHighlight) comment(c *drawutil.SyntaxHighlightComment) bool {
	pos0 := sh.sc.KeepPos()

	// must match sequence start (line or multiline)
	if err := sh.sc.M.Sequence(c.S); err != nil {
		return false
	}

	opt := &sh.d.Opt.SyntaxHighlight
	fg := opt.Comment.Fg
	bg := opt.Comment.Bg

	// single line comment
	if c.IsLine {
		op1 := &ColorizeOp{Offset: pos0.Pos, Fg: fg, Bg: bg}
		_ = sh.sc.M.ToNLExcludeOrEnd('\\')
		op2 := &ColorizeOp{Offset: sh.sc.Pos}
		sh.ops = append(sh.ops, op1, op2)
		return true
	}

	// multiline comment
	// start
	op := &ColorizeOp{Offset: pos0.Pos, Fg: fg, Bg: bg}
	sh.ops = append(sh.ops, op)
	// loop until it finds ending sequence
	for !sh.sc.M.Eof() {
		if err := sh.sc.M.Sequence(c.E); err == nil {
			// end
			op = &ColorizeOp{Offset: sh.sc.Pos}
			sh.ops = append(sh.ops, op)
			break
		}
		_, _ = sh.sc.ReadRune()
	}
	return true
}
