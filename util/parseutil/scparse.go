package parseutil

import (
	"fmt"
)

// scanner parse utility funcs
type ScParse struct {
	sc    *Scanner
	cache struct {
		cfs map[string]*ScParseCacheFn
	}
}

func (p *ScParse) init(sc *Scanner) {
	p.sc = sc
	p.cache.cfs = map[string]*ScParseCacheFn{}
}

//----------

func (p *ScParse) And(fns ...ScParseFn) ScParseFn {
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
func (p *ScParse) Or(fns ...ScParseFn) ScParseFn {
	return func() error {
		firstErr := error(nil)
		for _, fn := range fns {
			pos0 := p.sc.KeepPos()
			if err := fn(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				if IsScFatalError(err) {
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
func (p *ScParse) Optional(fn ScParseFn) ScParseFn {
	return func() error {
		pos0 := p.sc.KeepPos()
		if err := fn(); err != nil {
			if IsScFatalError(err) {
				return err
			}
			pos0.Restore()
			return nil
		}
		return nil
	}
}

//----------

func (p *ScParse) Loop(fn, sep ScParseFn, lastSep bool) ScParseFn {
	return func() error {
		sepPos := p.sc.KeepPos()
		for first := true; ; first = false {
			pos0 := p.sc.KeepPos()
			if err := fn(); err != nil {
				pos0.Restore()
				if IsScFatalError(err) {
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

func (p *ScParse) Rune(ru rune) ScParseFn {
	return func() error {
		return p.sc.M.Rune(ru)
	}
}
func (p *ScParse) RuneFn(fn func(rune) bool) ScParseFn {
	return func() error {
		return p.sc.M.RuneFn(fn)
	}
}
func (p *ScParse) Sequence(seq string) ScParseFn {
	return func() error {
		return p.sc.M.Sequence(seq)
	}
}

//----------

func (p *ScParse) DoubleQuotedString(maxLen int) ScParseFn {
	return func() error {
		return p.sc.M.DoubleQuotedString(maxLen)
	}
}
func (p *ScParse) QuotedString2(esc rune, maxLen1, maxLen2 int) ScParseFn {
	return func() error {
		return p.sc.M.QuotedString2(esc, maxLen1, maxLen2)
	}
}
func (p *ScParse) EscapeAny(esc rune) ScParseFn {
	return func() error {
		return p.sc.M.EscapeAny(esc)
	}
}
func (p *ScParse) Spaces(includeNL bool, escape rune) ScParseFn {
	return func() error {
		return p.sc.M.Spaces(includeNL, escape)
	}
}
func (p *ScParse) OptionalSpaces() ScParseFn {
	return p.Optional(p.Spaces(true, 0))
}
func (p *ScParse) Integer() ScParseFn {
	return p.sc.M.Integer
}
func (p *ScParse) Float() ScParseFn {
	return p.sc.M.Float
}

//----------

func (p *ScParse) GetCacheFunc(name string) *ScParseCacheFn {
	cf, ok := p.cache.cfs[name]
	if ok {
		return cf
	}
	cf = &ScParseCacheFn{p: p, name: name}
	p.cache.cfs[name] = cf
	return cf
}

//----------

func (p *ScParse) FatalOnErr(str string, fn ScParseFn) ScParseFn {
	return func() error {
		err := fn()
		if err != nil {
			if !IsScFatalError(err) {
				fe := &ScFatalError{}
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

// scanner parse cache func
type ScParseCacheFn struct {
	p    *ScParse
	name string
	Fn   ScParseFn
}

//----------
//----------
//----------

type ScParseFn func() error
