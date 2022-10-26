package lrparser

import (
	"errors"
	"io"
	"unicode"
	"unicode/utf8"
)

// parse state (used in grammarparser and contentparser)
type PState struct {
	src     []byte
	i       int
	reverse bool

	parseNode PNode
}

//----------

func (ps PState) copy() *PState {
	return &ps
}
func (ps *PState) set(ps2 *PState) {
	*ps = *ps2
}

//----------

func (ps *PState) readRune() (rune, error) {
	ru := rune(0)
	size := 0
	if ps.reverse {
		ru, size = utf8.DecodeLastRune(ps.src[:ps.i])
		size = -size // decrease ps.i
	} else {
		ru, size = utf8.DecodeRune(ps.src[ps.i:])
	}
	if size == 0 {
		return 0, io.EOF
	}
	ps.i += size
	return ru, nil
}

//----------

func (ps *PState) MatchRune(ru rune) error {
	ps2 := ps.copy()
	ru2, err := ps2.readRune()
	if err != nil {
		return err
	}
	if ru2 != ru {
		return errors.New("no match")
	}
	ps.set(ps2)
	return nil
}
func (ps *PState) MatchRunesOr(rs []rune) error {
	ps2 := ps.copy()
	ru, err := ps2.readRune()
	if err != nil {
		return err
	}
	if containsRune(rs, ru) {
		ps.set(ps2)
		return nil
	}
	return errors.New("no match")
}
func (ps *PState) MatchRunesAnd(rs []rune) error {
	ps2 := ps.copy()
	for i, l := 0, len(rs); i < l; i++ {
		ru := rs[i]
		if ps2.reverse {
			ru = rs[l-1-i]
		}
		ru2, err := ps2.readRune()
		if err != nil {
			return err
		}
		if ru2 != ru {
			return errors.New("no match")
		}
	}
	ps.set(ps2)
	return nil
}
func (ps *PState) matchRunesMid(rs []rune) error {
	ps2 := ps.copy()
	for k := 0; ; k++ {
		err := ps2.MatchRunesAnd(rs)
		if err == nil {
			ps.set(ps2)
			return nil
		}

		if k+1 >= len(rs) {
			break
		}

		// backup to previous rune to try to match again
		ps2.reverse = !ps.reverse
		if _, err := ps2.readRune(); err != nil {
			return err
		}
		ps2.reverse = ps.reverse
	}
	return errors.New("no match")
}
func (ps *PState) matchRunesNot(rs []rune) error {
	ps2 := ps.copy()
	ru, err := ps2.readRune()
	if err != nil {
		return err
	}
	if !containsRune(rs, ru) {
		ps.set(ps2)
		return nil
	}
	return errors.New("no match")
}

func (ps *PState) matchString(s string) error {
	return ps.MatchRunesAnd([]rune(s))
}

//----------

func (ps *PState) matchEof() error {
	ps2 := ps.copy()
	_, err := ps2.readRune()
	if errors.Is(err, io.EOF) {
		ps.set(ps2)
		return nil
	}
	return errors.New("no match")
}

//----------

func (ps *PState) consumeSpacesIncludingNL() bool {
	ps2 := ps.copy()
	for i := 0; ; i++ {
		ps3 := ps2.copy()
		ru, err := ps2.readRune()
		if err != nil {
			ps.set(ps2)
			return i > 0
		}
		if !unicode.IsSpace(ru) {
			ps.set(ps3)
			return i > 0
		}
	}
}
func (ps *PState) consumeSpacesExcludingNL() bool {
	ps2 := ps.copy()
	for i := 0; ; i++ {
		ps3 := ps2.copy()
		ru, err := ps2.readRune()
		if err != nil {
			ps.set(ps2)
			return i > 0
		}
		if !(unicode.IsSpace(ru) && ru != '\n') {
			ps.set(ps3)
			return i > 0
		}
	}
}

// allows escaped newlines
func (ps *PState) consumeSpacesExcludingNL2() bool {
	ok := false
	for {
		ok2 := ps.consumeSpacesExcludingNL()
		err := ps.matchString("\\\n")
		ok3 := err == nil
		if ok2 || ok3 {
			ok = true
		}
		if !ok2 && !ok3 {
			break
		}
	}
	return ok
}

func (ps *PState) consumeToNLIncluding() bool {
	ps2 := ps.copy()
	for i := 0; ; i++ {
		ps3 := ps2.copy()
		ru, err := ps2.readRune()
		if err != nil {
			ps.set(ps2)
			return i > 0
		}
		if ru == '\n' {
			ps.set(ps3)
			return true // include newline
		}
	}
}

//----------
//----------
//----------

type pstateParseFn func(ps *PState) error

//----------
//----------
//----------

func containsRune(rs []rune, ru rune) bool {
	for _, ru2 := range rs {
		if ru2 == ru {
			return true
		}
	}
	return false
}
