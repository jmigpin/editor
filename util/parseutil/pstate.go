package parseutil

import (
	"errors"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// parse state (used in lrparser grammarparser/contentparser)
type PState struct {
	Src     []byte
	Pos     int
	Reverse bool
	Node    PNode // parse node
}

func NewPState(src []byte) *PState {
	return &PState{Src: src}
}

//----------

func (ps PState) Copy() *PState {
	return &ps
}
func (ps *PState) Set(ps2 *PState) {
	*ps = *ps2
}

//----------

func (ps *PState) ReadRune() (rune, error) {
	ru := rune(0)
	size := 0
	if ps.Reverse {
		ru, size = utf8.DecodeLastRune(ps.Src[:ps.Pos])
		size = -size // decrease ps.i
	} else {
		ru, size = utf8.DecodeRune(ps.Src[ps.Pos:])
	}
	if size == 0 {
		return 0, io.EOF
	}
	ps.Pos += size
	return ru, nil
}

//----------

func (ps *PState) MatchRune(ru rune) error {
	ps2 := ps.Copy()
	ru2, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if ru2 != ru {
		return errors.New("no match")
	}
	ps.Set(ps2)
	return nil
}
func (ps *PState) MatchRunesOr(rs []rune) error {
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if ContainsRune(rs, ru) {
		ps.Set(ps2)
		return nil
	}
	return errors.New("no match")
}
func (ps *PState) MatchRunesAnd(rs []rune) error {
	ps2 := ps.Copy()
	for i, l := 0, len(rs); i < l; i++ {
		ru := rs[i]
		if ps2.Reverse {
			ru = rs[l-1-i]
		}
		ru2, err := ps2.ReadRune()
		if err != nil {
			return err
		}
		if ru2 != ru {
			return errors.New("no match")
		}
	}
	ps.Set(ps2)
	return nil
}
func (ps *PState) MatchRunesMid(rs []rune) error {
	ps2 := ps.Copy()
	for k := 0; ; k++ {
		err := ps2.MatchRunesAnd(rs)
		if err == nil {
			ps.Set(ps2)
			return nil
		}

		if k+1 >= len(rs) {
			break
		}

		// backup to previous rune to try to match again
		ps2.Reverse = !ps.Reverse
		if _, err := ps2.ReadRune(); err != nil {
			return err
		}
		ps2.Reverse = ps.Reverse
	}
	return errors.New("no match")
}
func (ps *PState) MatchRunesNot(rs []rune) error {
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if !ContainsRune(rs, ru) {
		ps.Set(ps2)
		return nil
	}
	return errors.New("no match")
}

func (ps *PState) MatchString(s string) error {
	return ps.MatchRunesAnd([]rune(s))
}

//----------

func (ps *PState) MatchEof() error {
	ps2 := ps.Copy()
	_, err := ps2.ReadRune()
	if errors.Is(err, io.EOF) {
		ps.Set(ps2)
		return nil
	}
	return errors.New("no match")
}

//----------

func (ps *PState) ConsumeSpacesIncludingNL() bool {
	ps2 := ps.Copy()
	for i := 0; ; i++ {
		ps3 := ps2.Copy()
		ru, err := ps2.ReadRune()
		if err != nil {
			ps.Set(ps2)
			return i > 0
		}
		if !unicode.IsSpace(ru) {
			ps.Set(ps3)
			return i > 0
		}
	}
}
func (ps *PState) ConsumeSpacesExcludingNL() bool {
	ps2 := ps.Copy()
	for i := 0; ; i++ {
		ps3 := ps2.Copy()
		ru, err := ps2.ReadRune()
		if err != nil {
			ps.Set(ps2)
			return i > 0
		}
		if !(unicode.IsSpace(ru) && ru != '\n') {
			ps.Set(ps3)
			return i > 0
		}
	}
}

// allows escaped newlines
func (ps *PState) ConsumeSpacesExcludingNL2() bool {
	ok := false
	for {
		ok2 := ps.ConsumeSpacesExcludingNL()
		err := ps.MatchString("\\\n")
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

func (ps *PState) ConsumeToNLIncluding() bool {
	ps2 := ps.Copy()
	for i := 0; ; i++ {
		ps3 := ps2.Copy()
		ru, err := ps2.ReadRune()
		if err != nil {
			ps.Set(ps2)
			return i > 0
		}
		if ru == '\n' {
			ps.Set(ps3)
			return true // include newline
		}
	}
}

//----------

// match opened/closed string sections.
func (ps *PState) StringSection(open, close string, escape rune, failOnNewline bool, maxLen int, eofClose bool) error {
	ps2 := ps.Copy()
	if err := ps2.MatchString(open); err != nil {
		return err
	}
	for {
		if escape != 0 && ps2.EscapeAny(escape) == nil {
			continue
		}
		if err := ps2.MatchString(close); err == nil {
			ps.Set(ps2)
			return nil // ok
		}
		// consume rune
		ru, err := ps2.ReadRune()
		if err != nil {
			// extension: stop on eof
			if eofClose && err == io.EOF {
				return nil // ok
			}

			return err
		}
		// extension: stop after maxlength
		if maxLen > 0 {
			d := ps2.Pos - ps.Pos
			if d < 0 { // handle reverse
				d = -d
			}
			if d > maxLen {
				return fmt.Errorf("passed maxlen")
			}
		}
		// extension: newline
		if failOnNewline && ru == '\n' {
			return fmt.Errorf("found newline")
		}
	}
}

//----------

func (ps *PState) GoString() error {
	return ps.GoString2('\\', 3000, 10)
}
func (ps *PState) GoString2(esc rune, maxLen1, maxLen2 int) error {
	q := "\"" // doublequote: fail on newline, eof doesn't close
	if err := ps.StringSection(q, q, esc, true, maxLen1, false); err == nil {
		return nil
	}
	q = "`" // backquote: can have newline, eof doesn't close
	if err := ps.StringSection(q, q, esc, false, maxLen1, false); err == nil {
		return nil
	}
	q = "'" // singlequote: fail on newline, eof doesn't close (usually a smaller maxlen)
	if err := ps.StringSection(q, q, esc, true, maxLen2, false); err == nil {
		return nil
	}
	return fmt.Errorf("not a string")
}

//----------

func (ps *PState) EscapeAny(escape rune) error {
	ps2 := ps.Copy()
	if ps.Reverse {
		if err := ps2.NRunes(1); err != nil {
			return err
		}
	}
	if err := ps2.MatchRune(escape); err != nil {
		return err
	}
	if !ps.Reverse {
		if err := ps2.NRunes(1); err != nil {
			return err
		}
	}
	ps.Set(ps2)
	return nil
}
func (ps *PState) NRunes(n int) error {
	ps2 := ps.Copy()
	for i := 0; i < n; i++ {
		_, err := ps2.ReadRune()
		if err != nil {
			return err
		}
	}
	ps.Set(ps2)
	return nil
}

//----------
//----------
//----------

// parse node
type PNode interface {
	Pos() int
	End() int
}

func PNodeBytes(node PNode, src []byte) []byte {
	pos, end := node.Pos(), node.End()
	if pos > end {
		pos, end = end, pos
	}
	return src[pos:end]
}
func PNodeString(node PNode, src []byte) string {
	return string(PNodeBytes(node, src))
}

func PNodePosStr(node PNode) string {
	return fmt.Sprintf("[%v:%v]", node.Pos(), node.End())
}
