package sampleparsers

import (
	"fmt"
	"unicode"

	"github.com/jmigpin/editor/util/parseutil/scanutil"
)

// A test/sample parser on how to use the smparse pkg.
type JsonParser struct {
	sc *scanutil.Scanner
	fn struct {
		object  scanutil.ParseFunc
		array   scanutil.ParseFunc
		element scanutil.ParseFunc
		member  scanutil.ParseFunc
		value   scanutil.ParseFunc
		number  scanutil.ParseFunc
	}
}

func NewJsonParser(src []byte) *JsonParser {
	p := &JsonParser{}
	p.sc = scanutil.NewScanner2(src)

	p.fn.object = p.sc.Parse.And(
		p.sc.Parse.Rune('{'),
		p.optionalSpaces(),
		p.sc.Parse.OptionalWithErr(-1, p.parseMembers),
		p.sc.Parse.Rune('}'),
	)
	p.fn.array = p.sc.Parse.And(
		p.sc.Parse.Rune('['),
		p.optionalSpaces(),
		p.sc.Parse.OptionalWithErr(-1, p.parseElements),
		p.sc.Parse.Rune(']'),
	)
	p.fn.number = p.sc.Parse.And(
		p.parseInteger,
		p.sc.Parse.OptionalNoErr(-1, p.parseFraction),
		p.sc.Parse.OptionalNoErr(-1, p.parseExponent),
	)
	p.fn.value = p.sc.Parse.Or(
		p.fn.object,
		p.fn.array,
		p.fn.number,
		p.parseString,
		p.sc.Parse.Sequence("true"),
		p.sc.Parse.Sequence("false"),
		p.sc.Parse.Sequence("null"),
	)
	p.fn.element = p.sc.Parse.And(
		p.optionalSpaces(),
		p.fn.value,
		p.optionalSpaces(),
	)
	p.fn.member = p.sc.Parse.And(
		p.optionalSpaces(),
		p.parseString,
		p.optionalSpaces(),
		p.sc.Parse.Rune(':'),
		p.fn.element,
	)
	return p
}

//----------

func (p *JsonParser) parseJson() (scanutil.Ast, error) {
	v, err := p.fn.element()
	if err == nil {
		if !p.sc.Match.End() {
			return nil, p.sc.Errorf("missing eof")
		}
	}
	return v, err
}

//----------

func (p *JsonParser) parseString() (scanutil.Ast, error) {
	return p.sc.Parse.DoubleQuotedString()
}

func (p *JsonParser) optionalSpaces() scanutil.ParseFunc {
	return p.sc.Parse.OptionalNoErr(-1, p.sc.Parse.Spaces)
}

//----------

func (p *JsonParser) parseMembers() (scanutil.Ast, error) {
	r := []scanutil.Ast{}
	for {
		v, err := p.fn.member()
		if err != nil {
			return nil, err
		}
		if v == nil {
			if len(r) >= 1 {
				return nil, fmt.Errorf("failed to parse member")
			}
			break
		}
		r = append(r, v)
		if p.sc.Match.Rune(',') {
			p.sc.Advance()
			continue
		}
		break
	}
	return r, nil
}

//----------

func (p *JsonParser) parseElements() (scanutil.Ast, error) {
	r := []scanutil.Ast{}
	for {
		v, err := p.fn.element()
		if err != nil {
			return nil, err
		}
		if v == nil {
			if len(r) >= 1 {
				return nil, fmt.Errorf("failed to parse member")
			}
			break
		}
		r = append(r, v)
		if p.sc.Match.Rune(',') {
			p.sc.Advance()
			continue
		}
		break
	}
	return r, nil
}

//----------

func (p *JsonParser) parseInteger() (scanutil.Ast, error) {
	r := p.sc.RewindOnFalse(func() bool {
		_ = p.sc.Match.Any("-")
		if p.sc.Match.Rune('0') {
			return true
		}
		return p.sc.Match.FnLoop(unicode.IsDigit)
	})
	if r {
		return p.sc.ValueAdv(), nil
	}
	return nil, nil
}

func (p *JsonParser) parseFraction() (scanutil.Ast, error) {
	r := p.sc.RewindOnFalse(func() bool {
		if !p.sc.Match.Rune('.') {
			return false
		}
		return p.sc.Match.FnLoop(unicode.IsDigit)
	})
	if r {
		return p.sc.ValueAdv(), nil
	}
	return nil, nil
}

func (p *JsonParser) parseExponent() (scanutil.Ast, error) {
	r := p.sc.RewindOnFalse(func() bool {
		if !p.sc.Match.Any("eE") {
			return false
		}
		_ = p.sc.Match.Any("+-")
		return p.sc.Match.FnLoop(unicode.IsDigit)
	})
	if r {
		return p.sc.ValueAdv(), nil
	}
	return nil, nil
}
