package drawer4

import (
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/statemach"
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
	sc  *statemach.Scanner
	ops []*ColorizeOp
}

func (sh *SyntaxHighlight) do(pad int) []*ColorizeOp {
	// limit reading to be able to handle big content
	o, n, _, _ := sh.d.visibleLen()
	min, max := o, o+n
	r := iorw.NewLimitedReader(sh.d.reader, min, max, pad)

	sh.sc = statemach.NewScanner(r)
	sh.sc.Advance()

	for !sh.sc.Match.End() {
		sh.normal(pad)
	}
	return sh.ops
}
func (sh *SyntaxHighlight) normal(pad int) {
	opt := &sh.d.Opt.SyntaxHighlight
	switch {
	case sh.sc.Match.Sequence(opt.Comment.Line.S):
		op1 := &ColorizeOp{
			Offset: sh.sc.Start,
			Fg:     opt.Comment.Line.Fg,
			Bg:     opt.Comment.Line.Bg,
		}
		sh.sc.Match.ToNewlineOrEnd()
		op2 := &ColorizeOp{Offset: sh.sc.Pos}
		sh.ops = append(sh.ops, op1, op2)
		sh.sc.Advance()
	case sh.sc.Match.Sequence(opt.Comment.Enclosed.S):
		// start
		op := &ColorizeOp{
			Offset: sh.sc.Start,
			Fg:     opt.Comment.Enclosed.Fg,
			Bg:     opt.Comment.Enclosed.Bg,
		}
		sh.ops = append(sh.ops, op)
		sh.sc.Advance()
		// loop until it finds ending sequence
		for !sh.sc.Match.End() {
			if sh.sc.Match.Sequence(opt.Comment.Enclosed.E) {
				// end
				op = &ColorizeOp{Offset: sh.sc.Pos}
				sh.ops = append(sh.ops, op)
				break
			}
			_ = sh.sc.ReadRune()
		}
		sh.sc.Advance()
	case sh.sc.Match.Quote('"', '\\', true, pad) ||
		sh.sc.Match.Quote('\'', '\\', true, 4):

		// unable to support multiline comments (Ex: Go backquotes) since the whole file is not parsed, just a section.
		// Also, in the case of Go backquotes, probably only .go files should support them.

		op1 := &ColorizeOp{
			Offset: sh.sc.Start,
			Fg:     opt.String.Fg,
			Bg:     opt.String.Bg,
		}
		op2 := &ColorizeOp{Offset: sh.sc.Pos}
		sh.ops = append(sh.ops, op1, op2)
		sh.sc.Advance()
	default:
		_ = sh.sc.ReadRune()
		sh.sc.Advance()
	}
}
