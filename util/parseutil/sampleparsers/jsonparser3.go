package sampleparsers

import (
	"strconv"

	"github.com/jmigpin/editor/util/parseutil/btparser"
)

func ParseJson3(src []byte) (any, error) {
	p := NewJsonParser3()
	return p.parseJson(src)
}

//----------

type JsonParser3 struct {
	g btparser.Rules
}

func NewJsonParser3() *JsonParser3 {
	return &JsonParser3{g: btparser.NewRules()}
}

func (p *JsonParser3) parseJson(src []byte) (any, error) {
	ps := btparser.NewParserStateFromBytes(src)

	valueFn := p.valueFn()
	optSpaces := p.g.Optional(p.g.Spaces())
	v := any(nil)
	_, err := p.g.Parse(ps, p.g.And(
		optSpaces,
		btparser.Assign(&v, valueFn),
		optSpaces,
		p.g.Eof(),
	))
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (p *JsonParser3) valueFn() btparser.VFn[any] {
	optSpaces := p.g.Optional(p.g.Spaces())
	memberFn := btparser.VFn[btparser.MapEntry[string, any]](nil)
	objectFn := btparser.VFn[map[string]any](nil)
	arrayFn := btparser.VFn[[]any](nil)
	valueFn := btparser.VFn[any](nil)

	stringFn := p.g.VQuotedString1()
	numberFn := p.numberFn()
	trueFn := btparser.VAny(btparser.VConst(p.g.Seq("true"), true))
	falseFn := btparser.VAny(btparser.VConst(p.g.Seq("false"), false))
	nullFn := btparser.VConst[any](p.g.Seq("null"), nil)

	memberFn = func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MapEntry[string, any], btparser.MPos, error) {
		m := btparser.MapEntry[string, any]{}
		mp, err := p.g.And(
			optSpaces,
			btparser.Assign(&m.Key, stringFn),
			optSpaces,
			p.g.Rune(':'),
			p.g.FatalOnError("member value", p.g.And(
				optSpaces,
				btparser.Assign(&m.Value, valueFn),
				optSpaces,
			)),
		)(ps, pos)
		return m, mp, err
	}
	objectFn = func(ps *btparser.ParserState, pos btparser.Pos) (map[string]any, btparser.MPos, error) {
		m := map[string]any{}
		mp, err := p.g.And(
			p.g.Rune('{'),
			optSpaces,
			p.g.Optional(p.g.LoopSep(
				true,
				btparser.SetMapEntry(&m, memberFn),
				p.g.And(
					optSpaces,
					p.g.Rune(','),
					optSpaces,
				),
			)),
			p.g.FatalOnError("expecting '}'", p.g.Rune('}')),
		)(ps, pos)
		return m, mp, err
	}
	arrayFn = func(ps *btparser.ParserState, pos btparser.Pos) ([]any, btparser.MPos, error) {
		w := []any{}
		mp, err := p.g.And(
			p.g.Rune('['),
			optSpaces,
			p.g.Optional(p.g.LoopSep(
				true,
				p.g.And(
					optSpaces,
					btparser.Append(&w, valueFn),
					optSpaces,
				),
				p.g.And(
					optSpaces,
					p.g.Rune(','),
					optSpaces,
				),
			)),
			p.g.FatalOnError("expecting ']'", p.g.Rune(']')),
		)(ps, pos)
		return w, mp, err
	}
	valueFn = btparser.VOr(
		btparser.VAny(objectFn),
		btparser.VAny(arrayFn),
		btparser.VAny(numberFn),
		btparser.VAny(stringFn),
		trueFn,
		falseFn,
		nullFn,
	)

	return valueFn
}

func (p *JsonParser3) numberFn() btparser.VFn[float64] {
	return func(ps *btparser.ParserState, pos btparser.Pos) (float64, btparser.MPos, error) {
		s, mp, err := p.g.VString(p.g.Or(
			p.g.Float(),
			p.g.Integer(),
		))(ps, pos)
		if err != nil {
			return 0, mp, err
		}
		v, err := strconv.ParseFloat(s, 64)
		return v, mp, err
	}
}

//----------
