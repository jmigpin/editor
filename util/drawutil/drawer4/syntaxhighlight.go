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

	maxDistBack := 5000
	sh := &SyntaxHighlight{d: d}
	d.Opt.SyntaxHighlight.Group.Ops = sh.do(maxDistBack)
}

//----------

type SyntaxHighlight struct {
	d   *Drawer
	sm  *statemach.SM
	ops []*ColorizeOp
}

func (sh *SyntaxHighlight) do(distBack int) []*ColorizeOp {
	// limit reading to be able to handle big content
	o, n, _, _ := sh.d.visibleLen()
	o -= distBack
	if o < 0 {
		distBack += o
		o = 0
	}
	n += distBack
	r := iorw.NewLimitedReaderLen(sh.d.reader, o, n)

	sh.sm = statemach.NewSM(r)
	sh.sm.Pos = o
	sh.sm.Advance()

	for !sh.sm.AcceptEnd() {
		sh.normal()
	}
	return sh.ops
}
func (sh *SyntaxHighlight) normal() {
	opt := &sh.d.Opt.SyntaxHighlight
	switch {
	case sh.sm.AcceptSequence(opt.Comment.Line.S):
		op1 := &ColorizeOp{
			Offset: sh.sm.Start,
			Fg:     opt.Comment.Line.Fg,
			Bg:     opt.Comment.Line.Bg,
		}
		sh.sm.AcceptToNewlineOrEnd()
		op2 := &ColorizeOp{Offset: sh.sm.Pos}
		sh.ops = append(sh.ops, op1, op2)
		sh.sm.Advance()
	case sh.sm.AcceptSequence(opt.Comment.Enclosed.S):
		// start
		op := &ColorizeOp{
			Offset: sh.sm.Start,
			Fg:     opt.Comment.Enclosed.Fg,
			Bg:     opt.Comment.Enclosed.Bg,
		}
		sh.ops = append(sh.ops, op)
		sh.sm.Advance()
		// loop until it finds ending sequence
		for !sh.sm.AcceptEnd() {
			if sh.sm.AcceptSequence(opt.Comment.Enclosed.E) {
				// end
				op = &ColorizeOp{Offset: sh.sm.Pos}
				sh.ops = append(sh.ops, op)
				break
			}
			_ = sh.sm.Next()
		}
		sh.sm.Advance()
	//case sh.sm.AcceptQuoteLoop("\"`'", "\\"):
	//	op1 := &ColorizeOp{
	//		Offset: sh.sm.Start,
	//		Fg:     opt.String.Fg,
	//		Bg:     opt.String.Bg,
	//	}
	//	op2 := &ColorizeOp{Offset: sh.sm.Pos}
	//	sh.ops = append(sh.ops, op1, op2)
	//	sh.sm.Advance()
	default:
		_ = sh.sm.Next()
		sh.sm.Advance()
	}
}
