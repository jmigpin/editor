package drawer4

import (
	"github.com/jmigpin/editor/util/statemach"
)

func updateSyntaxHighlightOps(d *Drawer, distBack int) {
	opt := &d.Opt.SyntaxHighlight
	opt.Group.Ops = SyntaxHighlightOps(d, distBack) // max distance back
}

func SyntaxHighlightOps(d *Drawer, distBack int) []*ColorizeOp {
	sh := &SyntaxHighlight{d: d}
	return sh.do(distBack)
}

//----------

type SyntaxHighlight struct {
	d   *Drawer
	sm  *statemach.String
	ops []*ColorizeOp
}

func (sh *SyntaxHighlight) do(distBack int) []*ColorizeOp {
	if !sh.d.Opt.RuneOffset.On {
		return nil
	}

	a := sh.d.Opt.RuneOffset.offset
	n := sh.d.runeOffsetViewLen()
	a -= distBack
	if a < 0 {
		distBack += a
		a = 0
	}
	n += distBack

	b, _ := sh.d.reader.ReadNSliceAt(a, n)
	bstr := string(b)

	sh.sm = statemach.NewString(bstr)
	for !sh.sm.AcceptRune(statemach.EOS) {
		sh.normal()
	}
	return sh.ops
}

func (sh *SyntaxHighlight) normal() {
	opt := &sh.d.Opt.SyntaxHighlight
	switch {
	case sh.sm.AcceptSequence(opt.Comment.Line.S):
		op := &ColorizeOp{
			Offset: sh.sm.Start,
			Fg:     opt.Comment.Line.Fg,
			Bg:     opt.Comment.Line.Bg,
		}
		sh.ops = append(sh.ops, op)
		if sh.sm.AcceptToNewlineOrEOS() {
			op := &ColorizeOp{Offset: sh.sm.Pos}
			sh.ops = append(sh.ops, op)
		}
		sh.sm.Advance()
	case sh.sm.AcceptSequence(opt.Comment.Enclosed.S):
		// start
		op := &ColorizeOp{
			Offset: sh.sm.Start,
			Fg:     opt.Comment.Enclosed.Fg,
			Bg:     opt.Comment.Enclosed.Bg,
		}
		sh.ops = append(sh.ops, op)
		// loop until it finds ending sequence
		for {
			if sh.sm.AcceptSequence(opt.Comment.Enclosed.E) {
				// end
				op = &ColorizeOp{Offset: sh.sm.Pos}
				sh.ops = append(sh.ops, op)
				break
			}
			if sh.sm.Next() == statemach.EOS {
				break
			}
		}
		sh.sm.Advance()
	case sh.sm.AcceptQuote("\"`'", "\\"): // TODO: max quote str size
		op := &ColorizeOp{
			Offset: sh.sm.Start,
			Fg:     opt.String.Fg,
			Bg:     opt.String.Bg,
		}
		sh.ops = append(sh.ops, op)
		op = &ColorizeOp{Offset: sh.sm.Pos}
		sh.ops = append(sh.ops, op)
		sh.sm.Advance()
	default:
		_ = sh.sm.Next()
		sh.sm.Advance()
	}
}
