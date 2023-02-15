package drawer4

import (
	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

func updateSyntaxHighlightOps(d *Drawer) {
	//if shDone(d) && phDone(d) {
	//	return
	//}
	if shDone(d) {
		return
	}

	sh := &SyntaxHighlight{d: d, pad: 4000}
	d.Opt.SyntaxHighlight.Group.Ops = sh.do()
}

//----------

func shDone(d *Drawer) bool {
	if !d.Opt.SyntaxHighlight.On {
		d.Opt.SyntaxHighlight.Group.Ops = nil
		return true
	}
	if d.opt.syntaxH.updated {
		return true
	}
	d.opt.syntaxH.updated = true
	return false
}

//func phDone(d *Drawer) bool {
//	if !d.Opt.ParenthesisHighlight.On {
//		d.Opt.ParenthesisHighlight.Group.Ops = nil
//		return true
//	}
//	if d.opt.parenthesisH.updated {
//		return true
//	}
//	d.opt.parenthesisH.updated = true
//	return false
//}

//----------

type SyntaxHighlight struct {
	d   *Drawer
	sc  *pscan.Scanner
	ops []*ColorizeOp
	pad int

	//// TODO: comments and strings parenthesis check
	//parens struct {
	//	pairs []rune
	//	w     []*parensPos
	//	main  *parensPos
	//}
}

func (sh *SyntaxHighlight) do() []*ColorizeOp {
	// limit reading to be able to handle big content
	o, n, _, _ := sh.d.visibleLen()
	min, max := o, o+n

	r := iorw.NewLimitedReaderAtPad(sh.d.reader, min, max, sh.pad)

	sh.sc = iorw.NewScanner(r)
	//sh.parens.pairs = []rune("{}()[]") // TODO: disabled

	_, _ = sh.sc.M.Loop(sh.sc.SrcMin(), sh.sc.W.Or(
		sh.parseString,
		sh.parseComment,
		//sh.parseParenthesis, // TODO: disabled
		sh.sc.M.OneRune,
	))

	//sh.processKeptParenthesis() // TODO: disabled

	return sh.ops
}

//----------

func (sh *SyntaxHighlight) parseComment(pos int) (int, error) {
	for _, c := range sh.d.Opt.SyntaxHighlight.Comment.Defs {
		if p2, err := sh.parseComment2(pos, c); err == nil {
			return p2, nil
		}
	}
	return pos, pscan.NoMatchErr
}
func (sh *SyntaxHighlight) parseComment2(pos int, c *drawutil.SyntaxHighlightComment) (int, error) {
	if p2, err := sh.sc.M.And(pos,
		sh.sc.W.Sequence(c.S),
		func(p3 int) (int, error) {
			// single line comment
			if c.IsLine {
				return sh.sc.M.ToNLOrErr(p3, false, '\\')

				// TODO
				//return sh.sc.W.OptLoop(p3,sh.sc.W.And(
				//	sh.sc.W.MustErr(sh.sc.W.Sequence(c.E)),
				//	sh.sc.M.OneRune,
				//)),
			}
			// multi line comment
			return sh.sc.M.And(p3,
				sh.sc.W.OptLoop(sh.sc.W.And(
					sh.sc.W.MustErr(sh.sc.W.Sequence(c.E)),
					sh.sc.M.OneRune,
				)),
				sh.sc.W.Sequence(c.E),
			)
		},
	); err != nil {
		return p2, err
	} else {
		opt := &sh.d.Opt.SyntaxHighlight
		fg := opt.Comment.Fg
		bg := opt.Comment.Bg

		op1 := &ColorizeOp{Offset: pos, Fg: fg, Bg: bg}
		op2 := &ColorizeOp{Offset: p2}
		sh.ops = append(sh.ops, op1, op2)

		return p2, nil
	}
}

//----------

func (sh *SyntaxHighlight) parseString(pos int) (int, error) {
	if p2, err := sh.sc.M.Or(pos,
		sh.sc.W.StringSection("\"", '\\', true, sh.pad, false),
		sh.sc.W.StringSection("'", '\\', true, 8, false), // consider '\x123'
	); err != nil {
		return p2, err
	} else {
		opt := &sh.d.Opt.SyntaxHighlight
		fg := opt.String.Fg
		bg := opt.String.Bg

		op1 := &ColorizeOp{Offset: pos, Fg: fg, Bg: bg}
		op2 := &ColorizeOp{Offset: p2}
		sh.ops = append(sh.ops, op1, op2)

		return p2, nil
	}
}

//----------

//func (sh *SyntaxHighlight) parseParenthesis(pos int) (int, error) {
//	if v, p2, err := sh.sc.M.RuneValue(pos,
//		sh.sc.W.RuneOneOf(sh.parens.pairs),
//	); err != nil {
//		return p2, err
//	} else {
//		// keep parenthesis position
//		ru := v.(rune)
//		pp := &parensPos{ru, p2}
//		sh.parens.w = append(sh.parens.w, pp)

//		switch p2 {
//		case sh.d.opt.cursor.offset, sh.d.opt.cursor.offset - 1:
//			sh.parens.main = pp
//		}
//		return p2, nil
//	}
//}

//func (sh *SyntaxHighlight) processKeptParenthesis() {
//	sh.d.Opt.ParenthesisHighlight.Group.Ops = nil
//	if sh.parens.main == nil {
//		return
//	}

//	// resolve open/close runes
//	sym := sh.parens.main.ru
//	k := strings.Index(string(sh.parens.pairs), string(sym))
//	isOpen := k%2 == 0
//	if isOpen {
//		k++
//	} else {
//		k--
//	}
//	openRu, closeRu := sym, sh.parens.pairs[k]

//	found := (*parensPos)(nil)
//	stk := 0
//	balanced := func(ru rune) bool {
//		switch ru {
//		case openRu:
//			stk++
//		case closeRu:
//			stk--
//			return stk == 0
//		}
//		return false
//	}

//	if isOpen {
//		for _, pp := range sh.parens.w {
//			if pp.pos < sh.parens.main.pos {
//				continue
//			}
//			if balanced(pp.ru) {
//				found = pp
//				break
//			}
//		}
//	} else {
//		for i := len(sh.parens.w) - 1; i >= 0; i-- {
//			pp := sh.parens.w[i]
//			if pp.pos > sh.parens.main.pos {
//				continue
//			}
//			if balanced(pp.ru) {
//				found = pp
//				break
//			}
//		}
//	}

//	// build points
//	points := []int{sh.parens.main.pos}
//	if found != nil {
//		points = append(points, found.pos)
//		if !isOpen {
//			points[0], points[1] = points[1], points[0]
//		}
//	}

//	// build colorize ops
//	opt := &sh.d.Opt.ParenthesisHighlight
//	fg := opt.Fg
//	bg := opt.Bg
//	ops := []*ColorizeOp{}
//	for _, p := range points {
//		op1 := &ColorizeOp{Offset: p, Fg: fg, Bg: bg}
//		op2 := &ColorizeOp{Offset: p + 1} // assumes rune size 1
//		ops = append(ops, op1, op2)
//	}
//	sh.d.Opt.ParenthesisHighlight.Group.Ops = ops
//}

////----------
////----------
////----------

//type parensPos struct {
//	ru  rune
//	pos int
//}
