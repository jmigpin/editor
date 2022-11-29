package parseutil

import "fmt"

// scanner parser utility funcs
type SParser struct {
	sc *Scanner
}

func (p *SParser) init(sc *Scanner) {
	p.sc = sc
}

//----------

func (p *SParser) And(fns ...SParserFunc) SParserFunc {
	return func() error {
		return p.sc.RestorePosOnErr(func() error {
			for _, fn := range fns {
				if err := fn(); err != nil {
					return err
				}
			}
			return nil
		})
	}
}
func (p *SParser) Or(fns ...SParserFunc) SParserFunc {
	return func() error {
		firstErr := error(nil)
		for _, fn := range fns {
			pos0 := p.sc.KeepPos()
			if err := fn(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				if IsFatalErr(err) {
					return err
				}
				pos0.Restore()
				continue
			}
			return nil
		}
		return firstErr
	}
}
func (p *SParser) Optional(fn SParserFunc) SParserFunc {
	return func() error {
		pos0 := p.sc.KeepPos()
		if err := fn(); err != nil {
			if IsFatalErr(err) {
				return err
			}
			pos0.Restore()
			return nil
		}
		return nil
	}
}

//----------

func (p *SParser) Loop(fn, sep SParserFunc, lastSep bool) SParserFunc {
	return func() error {
		sepPos := p.sc.KeepPos()
		for first := true; ; first = false {
			pos0 := p.sc.KeepPos()
			if err := fn(); err != nil {
				pos0.Restore()
				if IsFatalErr(err) {
					return err
				}
				if first {
					return err
				}
				if sep != nil && !first && !lastSep {
					sepPos.Restore()
					return fmt.Errorf("unexpected last separator")
				}
				return nil
			}
			if sep != nil {
				sepPos = p.sc.KeepPos()
				if err := sep(); err != nil {
					sepPos.Restore()
					return nil // no sep, last entry
				}
			}
		}
	}
}

func (p *SParser) Rune(ru rune) SParserFunc {
	return func() error {
		return p.sc.M.Rune(ru)
	}
}
func (p *SParser) Sequence(seq string) SParserFunc {
	return func() error {
		return p.sc.M.Sequence(seq)
	}
}

//----------

func (p *SParser) DoubleQuotedString(maxLen int) SParserFunc {
	return func() error {
		return p.sc.M.DoubleQuotedString(maxLen)
	}
}
func (p *SParser) Spaces(includeNL bool, escape rune) SParserFunc {
	return func() error {
		return p.sc.M.Spaces(includeNL, escape)
	}
}
func (p *SParser) OptionalSpaces() SParserFunc {
	return p.Optional(p.Spaces(true, 0))
}
func (p *SParser) Integer() SParserFunc {
	return p.sc.M.Integer
}
func (p *SParser) Float() SParserFunc {
	return p.sc.M.Float
}

//----------

func (p *SParser) FatalOnErr(str string, fn SParserFunc) SParserFunc {
	return func() error {
		err := fn()
		if err != nil {
			if !IsFatalErr(err) {
				fe := &SFatalError{}
				fe.error = fmt.Errorf("%v: %w", str, err)
				fe.Pos = p.sc.Pos
				err = fe
			}
		}
		return err
	}
}

//----------
//----------
//----------

type SParserFunc func() error
