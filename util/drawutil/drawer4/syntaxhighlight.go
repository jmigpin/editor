package drawer4

import (
	"image/color"

	"github.com/jmigpin/editor/util/drawutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/parseutil/btparser"
)

var syntaxHighlightP = newSyntaxHighlightParser()

func updateSyntaxHighlightOps(d *Drawer) {
	if shDone(d) {
		return
	}

	sh := &SyntaxHighlight{d: d, pad: syntaxHighlightPad}
	d.Opt.SyntaxHighlight.Group.Ops = sh.do()
}
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

//----------

type SyntaxHighlight struct {
	d   *Drawer
	pad int
}

func (sh *SyntaxHighlight) do() []*ColorizeOp {
	// limit reading to be able to handle big content
	o, n, _, _ := sh.d.visibleLen()
	min, max := o, o+n

	r := iorw.NewLimitedReaderAtPad(sh.d.reader, min, max, sh.pad)
	src, err := iorw.ReadFastFull(r)
	if err != nil {
		return nil
	}

	ps := btparser.NewParserStateFromBytes(src)
	data := &syntaxHighlightData{
		d:    sh.d,
		base: r.Min(),
	}
	ps.UserData[syntaxHighlightDataKey] = data

	syntaxHighlightP.parse(ps)

	return data.ops
}

//----------
//----------
//----------

type syntaxHighlightParser struct {
	g  btparser.Rules
	fn btparser.MFn
}

func newSyntaxHighlightParser() *syntaxHighlightParser {
	p := &syntaxHighlightParser{g: btparser.NewRules()}
	p.fn = p.build()
	return p
}

func (p *syntaxHighlightParser) parse(ps *btparser.ParserState) {
	_, _ = p.g.Parse(ps, p.fn)
}

func (p *syntaxHighlightParser) build() btparser.MFn {
	data := func(ps *btparser.ParserState) *syntaxHighlightData {
		data, ok := ps.UserData[syntaxHighlightDataKey].(*syntaxHighlightData)
		if !ok {
			panic("syntax highlight parser missing userdata")
		}
		return data
	}
	addOp := func(ps *btparser.ParserState, mp btparser.MPos, fg, bg color.Color) {
		data := data(ps)
		pos := int(mp.Start) + data.base
		p2 := int(mp.End) + data.base
		data.ops = append(data.ops,
			&ColorizeOp{Offset: pos, Fg: fg, Bg: bg},
			&ColorizeOp{Offset: p2},
		)
	}
	colorizeString := func(fn btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			mp, err := fn(ps, pos)
			if err != nil {
				return mp, err
			}
			data := data(ps)
			opt := &data.d.Opt.SyntaxHighlight
			addOp(ps, mp, opt.String.Fg, opt.String.Bg)
			return mp, nil
		}
	}
	colorizeComment := func(fn btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			mp, err := fn(ps, pos)
			if err != nil {
				return mp, err
			}
			data := data(ps)
			opt := &data.d.Opt.SyntaxHighlight
			addOp(ps, mp, opt.Comment.Fg, opt.Comment.Bg)
			return mp, nil
		}
	}
	commentSCs := func(makeFn func(*drawutil.SyntaxComment) btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			opt := &data(ps).d.Opt.SyntaxHighlight
			for _, c := range opt.Comment.SCs {
				if mp, err := makeFn(c)(ps, pos); err == nil {
					return mp, nil
				}
			}
			return btparser.MPos{Start: pos, End: pos}, btparser.NoMatchErr
		}
	}

	//----------

	lineComment := func(start string) btparser.MFn {
		return p.g.And(
			p.g.Seq(start),
			p.g.LoopToNLOrEof('\\', false),
		)
	}
	blockComment := func(start, end string) btparser.MFn {
		return p.g.Section(start, end, '\\', false, false, p.g.AnyRune())
	}
	stringFn := colorizeString(
		p.g.Or(
			p.g.WithBounds(0, syntaxHighlightPad, p.g.QuotedSection("\"", '\\', p.g.AnyExceptNewline())),
			p.g.WithBounds(0, 8, p.g.QuotedSection("'", '\\', p.g.AnyExceptNewline())),
		),
	)
	commentFn := colorizeComment(
		commentSCs(func(c *drawutil.SyntaxComment) btparser.MFn {
			if c.IsLine() {
				return lineComment(c.Start)
			}
			return blockComment(c.Start, c.End)
		}),
	)
	fn := p.g.Loop1(p.g.Or(
		stringFn,
		commentFn,
		p.g.AnyRune(),
	))

	return fn
}

//----------
//----------
//----------

const syntaxHighlightDataKey = "drawer4.syntaxhighlight.data"
const syntaxHighlightPad = 4000

type syntaxHighlightData struct {
	d    *Drawer
	base int
	ops  []*ColorizeOp
}
