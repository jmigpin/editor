package sampleparsers

import (
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

func ParseJson2(src []byte) (interface{}, error) {
	p := NewJsonParser2()
	return p.parseJson(src)
}

//----------

type JsonParser2 struct {
	sc *pscan.Scanner
	fn struct {
		doc      pscan.MFn
		object   pscan.MFn
		array    pscan.MFn
		value    pscan.MFn
		number   pscan.MFn
		string   pscan.MFn
		element  pscan.MFn
		elements pscan.MFn
		member   pscan.MFn
		members  pscan.MFn
	}

	testStrings []string
}

func NewJsonParser2() *JsonParser2 {
	p := &JsonParser2{}
	p.sc = pscan.NewScanner()

	// only defined later
	members := p.sc.W.PtrFn(&p.fn.members)
	elements := p.sc.W.PtrFn(&p.fn.elements)

	optSpaces := p.sc.W.Optional(p.sc.W.Spaces(true, 0))

	p.fn.object = p.sc.W.And(
		p.sc.W.Rune('{'),
		optSpaces,
		p.sc.W.Optional(members),
		p.sc.W.FatalOnError("expecting '}'",
			p.sc.W.Rune('}'),
		),
	)
	p.fn.array = p.sc.W.And(
		p.sc.W.Rune('['),
		optSpaces,
		p.sc.W.Optional(elements),
		p.sc.W.FatalOnError("expecting ']'",
			p.sc.W.Rune(']'),
		),
	)
	p.fn.number = p.sc.W.Or(
		p.sc.W.Float(),
		p.sc.W.Integer(),
	)
	p.fn.string = p.sc.W.DoubleQuotedString(3000)
	p.fn.value = p.sc.W.Or(
		p.fn.object,
		p.fn.array,
		p.fn.number,
		p.fn.string,
		p.sc.W.Sequence("true"),
		p.sc.W.Sequence("false"),
		p.sc.W.Sequence("null"),
	)
	p.fn.element = p.sc.W.And(
		optSpaces,
		p.fn.value,
		optSpaces,
	)
	p.fn.member = p.sc.W.And(
		optSpaces,
		p.fn.string,
		optSpaces,
		p.sc.W.Rune(':'),
		optSpaces,
		p.sc.W.FatalOnError("member element",
			p.fn.element,

			// TESTING
			//p.sc.W.OnValue(
			//	p.sc.W.BytesValue(p.fn.element),
			//	func(v any) {
			//		b, _ := v.([]byte)
			//		//fmt.Printf("%s\n", b)
			//		if len(b) > 0 && b[0] == '"' {
			//			p.testStrings = append(p.testStrings, string(b))
			//		}
			//	},
			//),
		),
	)
	p.fn.members = p.sc.W.LoopSepCanHaveLast(
		p.sc.W.FatalOnError("member",
			p.fn.member,
		),
		p.sc.W.Rune(','),
	)
	p.fn.elements = p.sc.W.LoopSepCanHaveLast(
		p.sc.W.FatalOnError("element",
			p.fn.element,
		),
		p.sc.W.Rune(','),
	)
	p.fn.doc = p.sc.W.And(
		p.fn.element,
		p.sc.M.Eof,
	)

	return p
}

func (p *JsonParser2) parseJson(src []byte) (any, error) {
	p.sc.SetSrc(src)
	if p2, err := p.fn.doc(0); err != nil {
		return nil, p.sc.SrcError(p2, err)
	} else {
		//return nil, nil // TODO: value
		return p.testStrings, nil // TESTING
	}
}

//----------
//----------
//----------

type SParserFunc = pscan.MFn
