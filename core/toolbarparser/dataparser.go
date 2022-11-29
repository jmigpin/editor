package toolbarparser

import (
	"fmt"
	"log"
	"unicode"

	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
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
	sc   *parseutil.Scanner
}

func newDataParser() *dataParser {
	p := &dataParser{}
	p.sc = parseutil.NewScanner()
	return p
}
func (p *dataParser) parse(src string) error {
	p.data = &Data{Str: src}
	p.sc.SetSrc([]byte(src))
	parts, err := p.parts()
	if err != nil {
		return err
	}
	p.data.Parts = parts
	return nil
}
func (p *dataParser) parts() ([]*Part, error) {
	parts := []*Part{}
	for {
		part, err := p.part()
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)

		// split parts on these runes
		if p.sc.M.RuneAny([]rune(("|\n"))) == nil {
			continue
		}
		if p.sc.M.Eof() {
			break
		}
	}
	return parts, nil
}
func (p *dataParser) part() (*Part, error) {
	pos0 := p.sc.KeepPos()

	_ = p.sc.M.SpacesExcludingNL() // optional space at start

	part := &Part{}
	part.Data = p.data
	for {
		arg, err := p.arg()
		if err != nil {
			break // end of part
		}
		part.Args = append(part.Args, arg)

		// need space between args
		if !p.sc.M.SpacesExcludingNL() {
			break
		}
	}

	part.SetPos(pos0.Pos, p.sc.Pos)
	return part, nil
}
func (p *dataParser) arg() (*Arg, error) {
	pos0 := p.sc.KeepPos()
	for {
		if p.sc.M.EscapeAny(osutil.EscapeRune) == nil {
			continue
		}
		if p.sc.M.QuotedString() == nil {
			continue
		}

		// split args
		pos3 := p.sc.KeepPos()
		ru, err := p.sc.ReadRune()
		if err == nil {
			valid := !(ru == '|' || unicode.IsSpace(ru))
			if !valid {
				err = parseutil.NoMatchErr
			}
		}
		if err != nil {
			pos3.Restore()
			break
		}
		// accept rune into arg
	}
	// empty arg. Ex: parts string with empty args: "|||".
	if pos0.IsEmpty() {
		return nil, fmt.Errorf("empty arg")
	}

	arg := &Arg{}
	arg.Data = p.data
	arg.SetPos(pos0.Pos, p.sc.Pos)
	return arg, nil
}
