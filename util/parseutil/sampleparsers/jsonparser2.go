package sampleparsers

import (
	"github.com/jmigpin/editor/util/parseutil"
)

func ParseJson2(src []byte) (interface{}, error) {
	p := NewJsonParser2()
	return p.parseJson(src)
}

//----------

type JsonParser2 struct {
	sc *parseutil.Scanner
	fn struct {
		object   parseutil.ScFn
		array    parseutil.ScFn
		value    parseutil.ScFn
		number   parseutil.ScFn
		string   parseutil.ScFn
		element  parseutil.ScFn
		elements parseutil.ScFn
		member   parseutil.ScFn
		members  parseutil.ScFn
	}
}

func NewJsonParser2() *JsonParser2 {
	p := &JsonParser2{}
	p.sc = parseutil.NewScanner()
	//p.sc.Debug = true

	// only defined later
	members := func() error {
		return p.fn.members()
	}
	elements := func() error {
		return p.fn.elements()
	}

	p.fn.object = p.sc.P.And(
		p.sc.P.Rune('{'),
		p.sc.P.OptionalSpaces(),
		p.sc.P.Optional(members),
		p.sc.P.FatalOnErr("expecting '}'",
			p.sc.P.Rune('}'),
		),
	)
	p.fn.array = p.sc.P.And(
		p.sc.P.Rune('['),
		p.sc.P.OptionalSpaces(),
		p.sc.P.Optional(elements),
		p.sc.P.FatalOnErr("expecting ']'",
			p.sc.P.Rune(']'),
		),
	)
	p.fn.number = p.sc.P.Or(
		p.sc.P.Float(),
		p.sc.P.Integer(),
	)
	p.fn.string = p.sc.P.DoubleQuotedString(3000)
	p.fn.value = p.sc.P.Or(
		p.fn.object,
		p.fn.array,
		p.fn.number,
		p.fn.string,
		p.sc.P.Sequence("true"),
		p.sc.P.Sequence("false"),
		p.sc.P.Sequence("null"),
	)
	p.fn.element = p.sc.P.And(
		p.sc.P.OptionalSpaces(),
		p.fn.value,
		p.sc.P.OptionalSpaces(),
	)
	p.fn.member = p.sc.P.And(
		p.sc.P.OptionalSpaces(),
		p.fn.string,
		p.sc.P.OptionalSpaces(),
		p.sc.P.Rune(':'),
		p.sc.P.FatalOnErr("member element",
			p.fn.element,
		),
	)
	p.fn.members = p.sc.P.Loop(
		p.sc.P.FatalOnErr("member",
			p.fn.member,
		),
		p.sc.P.Rune(','), true,
	)
	p.fn.elements = p.sc.P.Loop(
		p.sc.P.FatalOnErr("element",
			p.fn.element,
		),
		p.sc.P.Rune(','), true,
	)
	return p
}

func (p *JsonParser2) parseJson(src []byte) (any, error) {
	p.sc.SetSrc(src)
	if err := p.fn.element(); err != nil {
		return nil, p.sc.SrcError2(err, 50)
	}
	return nil, nil
}

//----------
//----------
//----------

type SParserFunc = parseutil.ScFn
