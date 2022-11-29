package scanutil

type Parse struct {
	sc *Scanner
}

//----------

func (p *Parse) And(fns ...ParseFunc) ParseFunc {
	return func() (Ast, error) {
		r := []Ast{}
		for _, fn := range fns {
			v, err := fn()
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, nil
			}
			r = append(r, v)
		}
		return r, nil
	}
}

func (p *Parse) Or(fns ...ParseFunc) ParseFunc {
	return func() (Ast, error) {
		for _, fn := range fns {
			v, err := fn()
			if err != nil {
				return nil, err
			}
			if v != nil {
				return v, nil
			}
		}
		return nil, nil
	}
}

func (p *Parse) OptionalWithErr(def Ast, fn ParseFunc) ParseFunc {
	return func() (Ast, error) {
		v, err := fn()
		if err != nil {
			return nil, err
		}
		if v == nil { // not parsed
			return def, nil
		}
		return v, nil
	}
}

func (p *Parse) OptionalNoErr(def Ast, fn ParseFunc) ParseFunc {
	return func() (Ast, error) {
		v, err := fn()
		if err != nil {
			return def, nil // ignore error
		}
		if v == nil { // not parsed
			return def, nil
		}
		return v, nil
	}
}

//----------

// consumes unrecognized input
func (p *Parse) LoopUntil(fn ParseFunc) ParseFunc {
	return func() (Ast, error) {
		for {
			v, err := fn()
			if err != nil {
				return nil, err
			}
			if v != nil {
				return v, nil
			}
			// consume unrecognized input
			ru := p.sc.ReadRune()
			if ru == Eof {
				break
			}
			p.sc.Advance()
		}
		return nil, nil
	}
}

func (p *Parse) RewindOnNilValue(fn ParseFunc) ParseFunc {
	return func() (v Ast, err error) {
		p.sc.RewindOnFalse(func() bool {
			v, err = fn()
			return v != nil
		})
		return
	}
}

//----------

func (p *Parse) ValueAdv(fn func() bool) ParseFunc {
	return func() (Ast, error) {
		if fn() {
			return p.sc.ValueAdv(), nil
		}
		return nil, nil
	}
}

//----------

func (p *Parse) Rune(ru rune) ParseFunc {
	return p.ValueAdv(func() bool {
		return p.sc.Match.Rune(ru)
	})
}

func (p *Parse) Sequence(s string) ParseFunc {
	return p.ValueAdv(func() bool {
		return p.sc.Match.Sequence(s)
	})
}

//----------

func (p *Parse) Spaces() (Ast, error) {
	if p.sc.Match.Spaces() {
		return p.sc.ValueAdv(), nil
	}
	return nil, nil
}

func (p *Parse) SpacesExceptNewline() (Ast, error) {
	if p.sc.Match.SpacesExceptNewline() {
		return p.sc.ValueAdv(), nil
	}
	return nil, nil
}

func (p *Parse) DoubleQuotedString() (Ast, error) {
	if p.sc.Match.DoubleQuoteStr() {
		return p.sc.ValueAdv(), nil
	}
	return nil, nil
}

//----------
//----------
//----------

type Ast interface{}

// returning (nil,nil) means not parsed
// returning (<somevalue>,nil) means parsed
type ParseFunc func() (Ast, error)
