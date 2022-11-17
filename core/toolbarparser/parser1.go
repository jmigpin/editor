package toolbarparser

//// DEPRECATED: old parser based on scanutil.scanner
//func parse1_basedOnScanutilScanner(str string) *Data {
//	p := &Parser{}
//	p.data = &Data{Str: str}

//	rd := iorw.NewStringReaderAt(str)
//	p.sc = scanutil.NewScanner(rd)

//	if err := p.start(); err != nil {
//		log.Print(err)
//	}
//	return p.data
//}

////----------

//type Parser struct {
//	data *Data
//	sc   *scanutil.Scanner
//}

//func (p *Parser) start() error {
//	parts, err := p.parts()
//	if err != nil {
//		return err
//	}
//	p.data.Parts = parts
//	return nil
//}

//func (p *Parser) parts() ([]*Part, error) {
//	var parts []*Part
//	for {
//		part, err := p.part()
//		if err != nil {
//			return nil, err
//		}
//		parts = append(parts, part)

//		// split parts on these runes
//		if p.sc.Match.Any("|\n") {
//			p.sc.Advance()
//			continue
//		}
//		if p.sc.Match.End() {
//			break
//		}
//	}
//	return parts, nil
//}

//func (p *Parser) part() (*Part, error) {
//	part := &Part{}
//	part.Data = p.data

//	// position
//	part.Pos = p.sc.Start
//	defer func() {
//		p.sc.Advance()
//		part.End = p.sc.Start
//	}()

//	// optional space at start
//	if p.sc.Match.SpacesExceptNewline() {
//		p.sc.Advance()
//	}

//	for {
//		arg, err := p.arg()
//		if err != nil {
//			break // end of part
//		}
//		part.Args = append(part.Args, arg)

//		// need space between args
//		if p.sc.Match.SpacesExceptNewline() {
//			p.sc.Advance()
//		} else {
//			break
//		}
//	}
//	return part, nil
//}

//func (p *Parser) arg() (*Arg, error) {
//	arg := &Arg{}
//	arg.Data = p.data

//	// position
//	arg.Pos = p.sc.Start
//	defer func() {
//		p.sc.Advance()
//		arg.End = p.sc.Start
//	}()

//	ok := p.sc.RewindOnFalse(func() bool {
//		for {
//			if p.sc.Match.End() {
//				break
//			}
//			if p.sc.Match.Escape(osutil.EscapeRune) {
//				continue
//			}
//			if p.sc.Match.GoQuotes(osutil.EscapeRune, 1500, 1500) {
//				continue
//			}

//			// split args
//			ru := p.sc.PeekRune()
//			if ru == '|' || unicode.IsSpace(ru) {
//				break
//			} else {
//				_ = p.sc.ReadRune() // accept rune into arg
//			}
//		}
//		return !p.sc.Empty()
//	})
//	if !ok {
//		// empty arg. Ex: parts string with empty args: "|||".
//		return nil, p.sc.Errorf("arg")
//	}
//	return arg, nil
//}
