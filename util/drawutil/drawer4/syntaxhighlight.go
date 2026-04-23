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
	sectionRules := buildSyntaxSectionRules(p.g, syntaxHighlightPad, func(ps *btparser.ParserState) []*drawutil.SyntaxComment {
		return data(ps).d.Opt.SyntaxHighlight.Comment.SCs
	})
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

	//----------

	stringFn := colorizeString(sectionRules.stringFn)
	commentFn := colorizeComment(sectionRules.commentFn)
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

//----------
//----------
//----------

type syntaxSectionRules struct {
	stringFn  btparser.MFn
	commentFn btparser.MFn
	anyFn     btparser.MFn
}

type syntaxSectionRange struct {
	start btparser.Pos
	end   btparser.Pos
}

// buildSyntaxSectionRules defines the string/comment section grammar used by syntax highlight colorization and parenthesis highlight section detection.
func buildSyntaxSectionRules(g btparser.Rules, maxStringLen int, syntaxComments func(*btparser.ParserState) []*drawutil.SyntaxComment) syntaxSectionRules {
	commentSCs := func(makeFn func(*drawutil.SyntaxComment) btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			for _, c := range syntaxComments(ps) {
				if mp, err := makeFn(c)(ps, pos); err == nil {
					return mp, nil
				}
			}
			return btparser.MPos{Start: pos, End: pos}, btparser.NoMatchErr
		}
	}

	//----------

	lineComment := func(start string) btparser.MFn {
		return g.And(
			g.Seq(start),
			g.LoopToNLOrEof('\\', false),
		)
	}
	blockComment := func(start, end string) btparser.MFn {
		return g.Section(start, end, '\\', false, false, g.AnyRune())
	}
	stringFn := g.Or(
		g.WithBounds(0, maxStringLen, g.QuotedSection("\"", '\\', g.AnyExceptNewline())),
		g.WithBounds(0, 8, g.QuotedSection("'", '\\', g.AnyExceptNewline())),
	)
	commentFn := commentSCs(func(c *drawutil.SyntaxComment) btparser.MFn {
		if c.IsLine() {
			return lineComment(c.Start)
		}
		return blockComment(c.Start, c.End)
	})

	return syntaxSectionRules{
		stringFn:  stringFn,
		commentFn: commentFn,
		anyFn:     g.Or(stringFn, commentFn),
	}
}
