package drawer4

import (
	"strings"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

func updateParenthesisHighlight(d *Drawer) {
	// TODO: testing handling parenthesis in syntaxhighlight
	//updateSyntaxHighlightOps(d)
	// return

	if !d.Opt.ParenthesisHighlight.On {
		d.Opt.ParenthesisHighlight.Group.Ops = nil
		return
	}

	if d.opt.parenthesisH.updated {
		return
	}
	d.opt.parenthesisH.updated = true

	ph := &ParenthesisHighlight{d: d, pad: 5000}
	d.Opt.ParenthesisHighlight.Group.Ops = ph.do()
}

//----------

type ParenthesisHighlight struct {
	d     *Drawer
	sc    *pscan.Scanner
	ops   []*ColorizeOp
	pad   int
	pairs []rune
}

func (ph *ParenthesisHighlight) do() []*ColorizeOp {
	ci := ph.d.opt.cursor.offset
	r := iorw.NewLimitedReaderAtPad(ph.d.reader, ci, ci, ph.pad)

	ph.sc = iorw.NewScanner(r)
	pos0 := ph.sc.ValidPos(ci)

	// match a parenthesis
	pairs := []rune("(){}[]")
	vk := ph.sc.NewValueKeeper()
	parseOpen := vk.WKeepValue(ph.sc.W.RuneValue(ph.sc.W.RuneOneOf(pairs)))
	_, err := parseOpen(pos0)
	if err != nil {
		//return nil // error: no results returned

		// try reading previous
		if p3, err2 := ph.sc.M.ReverseMode(pos0, true, parseOpen); err2 != nil {
			return nil // error: no results returned
		} else {
			pos0 = p3
		}
	}

	// pos0 is at the left side of the rune
	openPos := pos0

	// resolve open/close runes
	sym := vk.V.(rune)
	k := strings.Index(string(pairs), string(sym))
	isOpen := k%2 == 0
	if isOpen {
		k++
	} else {
		k--
	}
	openRu, closeRu := sym, pairs[k]
	reverse := !isOpen
	if reverse {
		pos0++ // to read the open rune again
	}

	// match parenthesis
	stk := 0
	done := false
	closePos := 0
	pushOpen := func(pos int) (int, error) {
		stk++
		return pos, nil
	}
	popClose := func(pos int) (int, error) {
		stk--
		if stk == 0 {
			done = true
			closePos = pos
			if !reverse {
				closePos--
			}
		}
		return pos, nil
	}
	_, _ = ph.sc.M.ReverseMode(pos0,
		reverse,
		ph.sc.W.Loop(ph.sc.W.And(
			ph.sc.W.PtrFalse(&done),
			ph.sc.W.Or(

				// might not work well (forward vs reverse)
				// ph.sc.W.QuotedString(),

				ph.sc.W.And(
					ph.sc.W.Rune(openRu),
					pushOpen,
				),
				ph.sc.W.And(
					ph.sc.W.Rune(closeRu),
					popClose,
				),
				ph.sc.M.OneRune,
			),
		)),
	)

	// sort points
	points := []int{openPos}
	hasClosePos := done == true
	if hasClosePos {
		points = append(points, closePos)
		if reverse {
			points[0], points[1] = points[1], points[0]
		}
	}

	// build colorize ops
	opt := &ph.d.Opt.ParenthesisHighlight
	fg := opt.Fg
	bg := opt.Bg
	for _, p := range points {
		op1 := &ColorizeOp{Offset: p, Fg: fg, Bg: bg}
		op2 := &ColorizeOp{Offset: p + 1} // assumes rune size 1
		ph.ops = append(ph.ops, op1, op2)
	}

	return ph.ops
}
