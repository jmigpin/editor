package drawer4

import (
	"strings"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil/btparser"
)

var parenthesisHighlightP = newParenthesisHighlightParser()

func updateParenthesisHighlight(d *Drawer) {
	if !d.Opt.ParenthesisHighlight.On {
		d.Opt.ParenthesisHighlight.Group.Ops = nil
		return
	}

	if d.opt.parenthesisH.updated {
		return
	}
	d.opt.parenthesisH.updated = true

	ph := &ParenthesisHighlight{d: d, pad: parenthesisHighlightPad}
	d.Opt.ParenthesisHighlight.Group.Ops = ph.do()
}

//----------

type ParenthesisHighlight struct {
	d   *Drawer
	pad int
}

func (ph *ParenthesisHighlight) do() []*ColorizeOp {
	ci := ph.d.opt.cursor.offset
	r := iorw.NewLimitedReaderAtPad(ph.d.reader, ci, ci, ph.pad)
	src, err := iorw.ReadFastFull(r)
	if err != nil {
		return nil
	}

	ps := btparser.NewParserStateFromBytes(src)
	data := &parenthesisHighlightData{
		d:    ph.d,
		base: r.Min(),
	}
	ps.UserData[parenthesisHighlightDataKey] = data

	pos := max(0, min(ci-data.base, len(src)))
	parenthesisHighlightP.parse(ps, btparser.Pos(pos))

	return data.ops
}

//----------
//----------
//----------

type parenthesisHighlightParser struct {
	g  btparser.Rules
	fn btparser.MFn
}

func newParenthesisHighlightParser() *parenthesisHighlightParser {
	p := &parenthesisHighlightParser{g: btparser.NewRules()}
	p.fn = p.build()
	return p
}

func (p *parenthesisHighlightParser) parse(ps *btparser.ParserState, pos btparser.Pos) {
	_, _ = p.g.ParseAt(ps, pos, p.fn)
}

func (p *parenthesisHighlightParser) build() btparser.MFn {
	pairs := []rune("(){}[]")
	data := func(ps *btparser.ParserState) *parenthesisHighlightData {
		data, ok := ps.UserData[parenthesisHighlightDataKey].(*parenthesisHighlightData)
		if !ok {
			panic("parenthesis highlight parser missing userdata")
		}
		return data
	}
	isPairRune := func(ru rune) bool {
		return strings.ContainsRune(string(pairs), ru)
	}
	vParen := func(fn btparser.VFn[rune]) btparser.VFn[parenthesisMatch] {
		return func(ps *btparser.ParserState, pos btparser.Pos) (parenthesisMatch, btparser.MPos, error) {
			ru, mp, err := fn(ps, pos)
			if err != nil {
				return parenthesisMatch{}, mp, err
			}
			if !isPairRune(ru) {
				return parenthesisMatch{}, btparser.MPos{Start: pos, End: pos}, btparser.NoMatchErr
			}
			return parenthesisMatch{ru: ru, mp: mp}, mp, nil
		}
	}
	resolvePair := func(ru rune) (rune, rune, bool) {
		k := strings.Index(string(pairs), string(ru))
		isOpen := k%2 == 0
		if isOpen {
			return ru, pairs[k+1], false
		}
		return ru, pairs[k-1], true
	}
	addOps := func(ps *btparser.ParserState, openPos int, closePos int, hasClosePos bool, reverse bool) {
		points := []int{openPos}
		if hasClosePos {
			points = append(points, closePos)
			if reverse {
				points[0], points[1] = points[1], points[0]
			}
		}

		data := data(ps)
		opt := &data.d.Opt.ParenthesisHighlight
		for _, p := range points {
			data.ops = append(data.ops,
				&ColorizeOp{Offset: p, Fg: opt.Fg, Bg: opt.Bg},
				&ColorizeOp{Offset: p + 1}, // assumes rune size 1 like the previous implementation
			)
		}
	}

	//----------

	scanPair := func(ps *btparser.ParserState, pos btparser.Pos, reverse bool, openRu, closeRu rune) (int, bool) {
		stk := 0
		for {
			ru := rune(0)
			mp := btparser.MPos{}
			err := error(nil)
			if reverse {
				ru, mp, err = p.g.VLastRune()(ps, pos)
			} else {
				ru, mp, err = p.g.VRune()(ps, pos)
			}
			if err != nil {
				return 0, false
			}
			pos = mp.End

			// TODO: quoted strings/comments are not skipped yet.
			switch ru {
			case openRu:
				stk++
			case closeRu:
				stk--
				if stk == 0 {
					p, _ := mp.Bounds()
					return int(p) + data(ps).base, true
				}
			}
		}
	}

	//----------

	startParen := btparser.VOr(
		vParen(p.g.VRune()),
		vParen(p.g.VLastRune()),
	)
	fn := func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
		pm, mp, err := startParen(ps, pos)
		if err != nil {
			return mp, err
		}

		openStart, openEnd := pm.mp.Bounds()
		openPos := int(openStart) + data(ps).base
		openRu, closeRu, reverse := resolvePair(pm.ru)
		if reverse {
			pos = openEnd // read the open rune again
		} else {
			pos = openStart
		}

		closePos, done := scanPair(ps, pos, reverse, openRu, closeRu)
		addOps(ps, openPos, closePos, done, reverse)
		return btparser.MPos{Start: pos, End: pos}, nil
	}

	return fn
}

//----------
//----------
//----------

const parenthesisHighlightDataKey = "drawer4.parenthesishighlight.data"
const parenthesisHighlightPad = 5000

type parenthesisHighlightData struct {
	d    *Drawer
	base int
	ops  []*ColorizeOp
}

type parenthesisMatch struct {
	ru rune
	mp btparser.MPos
}
