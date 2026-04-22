package toolbarparser

import (
	"log"
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/btparser"
)

const dataParserDataKey = "toolbarparser.data"

func Parse(str string) *Data {
	p := getDataParserRules()
	data := &Data{Str: str}
	ps := btparser.NewParserStateFromString(str)
	ps.UserData[dataParserDataKey] = data

	if _, err := p.g.Parse(ps, p.fn); err != nil {
		log.Print(err)
	}
	return data
}

//----------
//----------
//----------

type dataParserRules struct {
	g  btparser.Rules
	fn btparser.MFn
}

var dataParserRules1 *dataParserRules

func getDataParserRules() *dataParserRules {
	if dataParserRules1 == nil {
		dataParserRules1 = initDataParserRules()
	}
	return dataParserRules1
}

func initDataParserRules() *dataParserRules {
	p := &dataParserRules{}
	p.g = btparser.NewRules()
	g := p.g

	//----------

	data := func(ps *btparser.ParserState) *Data {
		data, ok := ps.UserData[dataParserDataKey].(*Data)
		if !ok {
			panic("toolbar parser missing Data userdata")
		}
		return data
	}
	newArg := func(ps *btparser.ParserState, mp btparser.MPos) *Arg {
		arg := &Arg{}
		arg.Data = data(ps)
		arg.SetPos(int(mp.Start), int(mp.End))
		return arg
	}
	appendArg := func(part *Part, fn btparser.MFn) btparser.MFn {
		return btparser.AppendLocal(&part.Args, btparser.VFromMPos(fn, newArg))
	}
	newPartRule := func(body func(*Part) btparser.MFn) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			data := data(ps)
			part := &Part{}
			part.Data = data

			mp, err := body(part)(ps, pos)
			if err != nil {
				return mp, err
			}

			part.SetPos(int(pos), int(mp.End))
			data.Parts = append(data.Parts, part)
			return btparser.MPos{Start: pos, End: mp.End}, nil
		}
	}

	//----------

	quotedString := g.QuotedString2(osutil.EscapeRune, 3000, 8)
	spaces := g.Loop1(g.Or(
		g.And(
			g.Rune(osutil.EscapeRune),
			g.RuneFn(unicode.IsSpace),
		),
		g.RuneFn(func(ru rune) bool {
			return unicode.IsSpace(ru) && ru != '\n'
		}),
	))
	arg := g.Loop1(g.Or(
		g.Escape(osutil.EscapeRune),
		quotedString,
		g.RuneFn(func(ru rune) bool {
			return ru != '|' && !unicode.IsSpace(ru)
		}),
	))
	part := newPartRule(func(part *Part) btparser.MFn {
		return g.Optional(g.Loop1(g.Or(
			spaces,
			appendArg(part, arg),
		)))
	})
	parts := g.LoopSepAllowEmpty(
		part,
		g.RuneAnyOf('|', '\n'),
	)
	p.fn = g.And(
		parts,
		g.Eof(),
	)
	return p
}
