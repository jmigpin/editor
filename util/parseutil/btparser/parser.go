package btparser

import (
	"errors"
	"fmt"
)

type Parser struct {
	src []byte

	tokenC int
	ignore struct {
		fn    MFn
		c     int // recursive call count (avoid loops)
		cache struct {
			valid  bool
			pos    Pos
			result Pos
		}
	}

	farthest Pos
}

func NewParser() *Parser {
	p := &Parser{}
	return p
}

func (p *Parser) G() Rules {
	return NewRules(p)
}

func (p *Parser) SetSrc(src []byte) {
	p.src = src
}
func (p *Parser) SetSrcFromString(s string) {
	p.src = []byte(s)
}

//----------

func (p *Parser) Parse(fn MFn) (Pos, error) {
	mp, err := fn(0)
	if err != nil {
		err = fmt.Errorf("%v: %q", err, p.Snippet(mp))

		mp2 := MPos{p.farthest, p.farthest}
		if mp != mp2 {
			err2 := fmt.Errorf("farthest: %q", p.Snippet(mp2))
			err = errors.Join(err, err2)
		}

		return mp.End, err
	}
	return mp.End, nil
}

//----------

func (p *Parser) SetIgnore(fn MFn) {
	p.ignore.fn = fn
}

func (p *Parser) runIgnore(pos Pos) Pos {
	// don't run inside a token
	if p.tokenC > 0 {
		return pos
	}

	// avoid endless loop: ignore fns can themselves trigger runIgnores
	p.ignore.c++
	defer func() { p.ignore.c-- }()
	if p.ignore.c > 1 {
		return pos
	}

	if p.ignore.fn != nil {
		if p.ignore.cache.valid && p.ignore.cache.pos == pos {
			// DEBUG
			//u := p.ignore.cache.result - p.ignore.cache.pos
			//fmt.Printf("using ignore cache: %v\n", u)

			return p.ignore.cache.result
		}
		if mp, err := p.ignore.fn(pos); err == nil {
			p.ignore.cache.valid = true
			p.ignore.cache.pos = pos
			p.ignore.cache.result = mp.End

			// DEBUG
			//fmt.Printf("ignored: %q\n", p.Source(mp))

			pos = mp.End
		}
	}

	return pos
}

//----------

func (p *Parser) Source(mp MPos) []byte {
	return p.src[mp.Start:mp.End]
}
func (p *Parser) SourceStr(mp MPos) string {
	return string(p.Source(mp))
}
func (p *Parser) Snippet(mp MPos) string {
	return BytesSnippet(p.src, mp, 30)
}

//----------
//----------
//----------

type Pos int

type MPos struct { // match pos
	Start Pos
	End   Pos
}

//----------

// match/value funcs
// MFn->VFn: MHandleMFn
// VFn->MFn: MHandleVFn

type MFn func(Pos) (MPos, error)
type VFn[T any] func(Pos) (T, MPos, error) //

type VHandler[T any] func(T) error       //
type VMaker[T any] func(MPos) (T, error) //

//----------

func BoolErrFn[T any](fn func(T) bool) func(T) error {
	return func(v T) error {
		if fn(v) {
			return nil
		}
		return NoMatchErr
	}
}

//----------

var NoMatchErr = fmt.Errorf("no match")
