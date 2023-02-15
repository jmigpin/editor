package toolbarparser

import (
	"log"
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil/pscan"
)

func Parse(str string) *Data {
	p := newDataParser()
	if err := p.parse(str); err != nil {
		log.Print(err)
	}
	return p.data
}

//----------
//----------
//----------

type dataParser struct {
	data *Data
	sc   *pscan.Scanner
}

func newDataParser() *dataParser {
	p := &dataParser{}
	p.sc = pscan.NewScanner()
	return p
}
func (p *dataParser) parse(src string) error {
	p.data = &Data{Str: src}
	p.sc.SetSrc([]byte(src))

	vk := p.sc.NewValueKeeper()
	if p2, err := p.sc.M.And(0,
		vk.WKeepValue(p.parseParts),
		p.sc.M.Eof,
	); err != nil {
		return p.sc.SrcError(p2, err)
	}

	p.data.Parts = vk.V.([]*Part)
	return nil
}
func (p *dataParser) parseParts(pos int) (any, int, error) {
	parts := []*Part{}
	p2, err := p.sc.M.LoopSepCanHaveLast(pos,
		p.sc.W.OnValue(
			p.parsePart,
			func(v any) { parts = append(parts, v.(*Part)) },
		),
		// separator
		p.sc.W.RuneOneOf([]rune("|\n")),
	)
	return parts, p2, err
}
func (p *dataParser) parsePart(pos int) (any, int, error) {
	part := &Part{}
	part.Data = p.data

	// optloop: arg can be nil
	p2, err := p.sc.M.OptLoop(pos, p.sc.W.Or(
		p.parseSpaces,
		p.sc.W.OnValue(
			p.parseArg,
			func(v any) { part.Args = append(part.Args, v.(*Arg)) },
		),
	))
	// NOTE: should never be an error with optloop, still leaving it here
	if err != nil {
		return nil, p2, err
	}

	part.SetPos(pos, p2)
	return part, p2, nil
}
func (p *dataParser) parseArg(pos int) (any, int, error) {
	argRune := func(ru rune) bool {
		return ru != '|' && !unicode.IsSpace(ru)
	}
	if p2, err := p.sc.M.Loop(pos, p.sc.W.Or(
		p.sc.W.EscapeAny(osutil.EscapeRune),
		p.sc.W.QuotedString(),
		p.sc.W.RuneFn(argRune),
	)); err != nil {
		return nil, p2, err
	} else {
		arg := &Arg{}
		arg.Data = p.data
		arg.SetPos(pos, p2)
		return arg, p2, nil
	}
}
func (p *dataParser) parseOptSpaces(pos int) (int, error) {
	return p.sc.M.Optional(pos, p.parseSpaces)
}
func (p *dataParser) parseSpaces(pos int) (int, error) {
	return p.sc.M.Spaces(pos, false, '\\')
}
