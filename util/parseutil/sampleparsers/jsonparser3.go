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

const jsonValueKey = "sampleparsers.json3.value"

func NewJsonParser3() *JsonParser3 {
	return &JsonParser3{g: btparser.NewRules()}
}

func (p *JsonParser3) parseJson(src []byte) (any, error) {
	ps := btparser.NewParserStateFromBytes(src)
	v := any(nil)
	ps.UserData[jsonValueKey] = &v

	_, err := p.g.Parse(ps, p.build())
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (p *JsonParser3) build() btparser.MFn {
	optSpaces := p.g.Optional(p.g.Spaces())
	memberFn := btparser.VFn[jsonMember](nil)
	objectFn := btparser.VFn[map[string]any](nil)
	arrayFn := btparser.VFn[[]any](nil)
	valueFn := btparser.VFn[any](nil)

	setMember := func(m map[string]any, fn btparser.VFn[jsonMember]) btparser.MFn {
		return func(ps *btparser.ParserState, pos btparser.Pos) (btparser.MPos, error) {
			member, mp, err := fn(ps, pos)
			if err != nil {
				return mp, err
			}
			m[member.key] = member.value
			return mp, nil
		}
	}
	number := func(fn btparser.MFn) btparser.VFn[float64] {
		return func(ps *btparser.ParserState, pos btparser.Pos) (float64, btparser.MPos, error) {
			s, mp, err := p.g.VString(fn)(ps, pos)
			if err != nil {
				return 0, mp, err
			}
			v, err := strconv.ParseFloat(s, 64)
			return v, mp, err
		}
	}
	valueData := btparser.UserDataPtrFn[any](jsonValueKey)

	//----------

	stringFn := p.g.VQuotedString1()
	numberFn := number(p.g.Or(
		p.g.Float(),
		p.g.Integer(),
	))
	trueFn := btparser.VAny(btparser.VConst(p.g.Seq("true"), true))
	falseFn := btparser.VAny(btparser.VConst(p.g.Seq("false"), false))
	nullFn := btparser.VConst[any](p.g.Seq("null"), nil)
	memberFn = func(ps *btparser.ParserState, pos btparser.Pos) (jsonMember, btparser.MPos, error) {
		m := jsonMember{}
		mp, err := p.g.And(
			optSpaces,
			btparser.AssignLocal(&m.key, stringFn),
			optSpaces,
			p.g.Rune(':'),
			p.g.FatalOnError("member value", p.g.And(
				optSpaces,
				btparser.AssignLocal(&m.value, valueFn),
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
				setMember(m, memberFn),
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
					btparser.AppendLocal(&w, valueFn),
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
	fn := p.g.And(
		optSpaces,
		btparser.AssignFn(valueData, valueFn),
		optSpaces,
		p.g.Eof(),
	)

	return fn
}

//----------

type jsonMember struct {
	key   string
	value any
}

//----------
