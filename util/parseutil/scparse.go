package parseutil

import (
	"fmt"
)

// scanner parse utility funcs
type ScParse struct {
	sc    *Scanner
	M     *ScMatch
	cache struct {
		cfs map[string]*ScParseCacheFn
	}
}

func (p *ScParse) init(sc *Scanner) {
	p.sc = sc
	p.M = &sc.M
	p.cache.cfs = map[string]*ScParseCacheFn{}
}

//----------

func (p *ScParse) And(fns ...ScFn) ScFn {
	return func() error {
		return p.M.And(fns...)
	}
}
func (p *ScParse) Or(fns ...ScFn) ScFn {
	return func() error {
		return p.M.Or(fns...)
	}
}
func (p *ScParse) Optional(fn ScFn) ScFn {
	return func() error {
		return p.M.Optional(fn)
	}
}

//----------

func (p *ScParse) Loop(fn, sep ScFn, lastSep bool) ScFn {
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

func (p *ScParse) Rune(ru rune) ScFn {
	return func() error {
		return p.M.Rune(ru)
	}
}
func (p *ScParse) RuneAny(rs []rune) ScFn {
	return func() error {
		return p.M.RuneAny(rs)
	}
}
func (p *ScParse) RuneFn(fn func(rune) bool) ScFn {
	return func() error {
		return p.M.RuneFn(fn)
	}
}
func (p *ScParse) Sequence(seq string) ScFn {
	return func() error {
		return p.M.Sequence(seq)
	}
}

//----------

func (p *ScParse) RegexpFromStartCached(res string, maxLen int) ScFn {
	return func() error {
		return p.M.RegexpFromStartCached(res, maxLen)
	}
}
func (p *ScParse) DoubleQuotedString(maxLen int) ScFn {
	return func() error {
		return p.M.DoubleQuotedString(maxLen)
	}
}
func (p *ScParse) QuotedString2(esc rune, maxLen1, maxLen2 int) ScFn {
	return func() error {
		return p.M.QuotedString2(esc, maxLen1, maxLen2)
	}
}
func (p *ScParse) EscapeAny(esc rune) ScFn {
	return func() error {
		return p.M.EscapeAny(esc)
	}
}
func (p *ScParse) NRunes(n int) ScFn {
	return func() error {
		return p.M.NRunes(n)
	}
}
func (p *ScParse) Spaces(includeNL bool, escape rune) ScFn {
	return func() error {
		return p.M.Spaces(includeNL, escape)
	}
}
func (p *ScParse) OptionalSpaces() ScFn {
	return p.Optional(p.Spaces(true, 0))
}
func (p *ScParse) Integer() ScFn {
	return p.M.Integer
}
func (p *ScParse) Float() ScFn {
	return p.M.Float
}

//----------

func (p *ScParse) NewValueKeeper() *ScValueKeeper {
	ak := &ScValueKeeper{p: p}
	return ak
}

//----------

// WARNING: best used when there are no closure variables in the function, otherwise the variables will contain values of previous runs
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

func (p *ScParse) FatalOnErr(str string, fn ScFn) ScFn {
	return func() error {
		err := fn()
		if err != nil {
			if !IsScFatalError(err) {
				fe := &ScFatalError{}
				fe.Err = fmt.Errorf("%v: %w", str, err)
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
	fn   ScFn

	PreRun func()
	Data   func() any
}

func (cf *ScParseCacheFn) IsSet() bool {
	return cf.fn != nil
}
func (cf *ScParseCacheFn) Set(fn ScFn) {
	cf.fn = fn
}
func (cf *ScParseCacheFn) Run() error {
	if cf.PreRun != nil {
		cf.PreRun()
	}
	return cf.fn()
}

//----------
//----------
//----------

type ScValueKeeper struct {
	p     *ScParse
	Value any
}

func (vk *ScValueKeeper) Keep(fn ScValueFn) ScFn {
	return func() error {
		v, err := fn()
		vk.Value = v
		return err
	}
}
func (vk *ScValueKeeper) KeepBytes(fn ScFn) ScFn {
	return func() error {
		pos0 := vk.p.sc.KeepPos()
		err := fn()
		vk.Value = pos0.Bytes()
		return err
	}
}
func (vk *ScValueKeeper) StringOptional() string {
	if vk.Value == nil {
		return ""
	}
	return vk.Value.(string)
}
func (vk *ScValueKeeper) String() string {
	return vk.Value.(string)
}
func (vk *ScValueKeeper) Bytes() []byte {
	return vk.Value.([]byte)
}

//----------
//----------
//----------

type ScFn func() error
type ScValueFn func() (any, error)
