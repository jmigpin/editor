package statemach

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

const eos = -1
const readErr = -2

type SM struct {
	Start int
	Pos   int
	r     iorw.Reader
}

func NewSM(r iorw.Reader) *SM {
	return &SM{r: r}
}

func (sm *SM) Next() rune {
	if sm.Pos >= sm.r.Len() {
		return eos
	}
	ru, w, err := sm.r.ReadRuneAt(sm.Pos)
	if err != nil {
		return readErr
	}
	sm.Pos += w
	return ru
}

func (sm *SM) Peek() rune {
	p := sm.Pos
	u := sm.Next()
	sm.Pos = p
	return u
}

func (sm *SM) Advance() {
	sm.Start = sm.Pos
}

//----------

func (sm *SM) AcceptRune(ru rune) bool {
	pos := sm.Pos
	if sm.Next() == ru {
		return true
	}
	sm.Pos = pos
	return false
}

// The end value can be obtained with next() or peek().
func (sm *SM) AcceptEnd() bool {
	return sm.AcceptRune(eos) || sm.AcceptRune(readErr)
}

//----------

func (sm *SM) AcceptAny(valid string) bool {
	pos := sm.Pos
	if strings.ContainsRune(valid, sm.Next()) {
		return true
	}
	sm.Pos = pos
	return false
}

func (sm *SM) AcceptAnyNeg(invalid string) bool {
	pos := sm.Pos
	if !strings.ContainsRune(invalid, sm.Next()) {
		return true
	}
	sm.Pos = pos
	return false
}

func (sm *SM) AcceptSequence(s string) bool {
	if s == "" {
		return false
	}
	p := sm.Pos
	for _, ru := range s {
		if ru != sm.Next() {
			sm.Pos = p
			return false
		}
	}
	return true
}

func (sm *SM) AcceptFn(fn func(rune) bool) bool {
	p := sm.Pos
	if fn(sm.Next()) {
		return true
	}
	sm.Pos = p
	return false
}

func (sm *SM) AcceptLoop(valid string) bool {
	v := false
	for sm.AcceptAny(valid) {
		v = true
	}
	return v
}

func (sm *SM) AcceptLoopFn(fn func(rune) bool) bool {
	v := false
	for !sm.AcceptEnd() {
		if sm.AcceptFn(fn) {
			v = true // getting at least one will return true
			continue
		}
		break
	}
	return v

}

func (sm *SM) AcceptN(n int) bool {
	if sm.Pos+n > sm.r.Len() {
		return false
	}
	sm.Pos += n
	return true
}

func (sm *SM) AcceptNRunes(n int) bool {
	p := sm.Pos
	_ = sm.AcceptLoopFn(func(ru rune) bool {
		if n <= 0 {
			return false
		}
		n--
		return true
	})
	if n <= 0 {
		return true
	}
	sm.Pos = p
	return false
}

//----------

func (sm *SM) AcceptSpace() bool {
	return sm.AcceptLoopFn(unicode.IsSpace)
}

func (sm *SM) AcceptSpaceExceptNewline() bool {
	return sm.AcceptLoopFn(func(ru rune) bool {
		if ru == '\n' {
			return false
		}
		return unicode.IsSpace(ru)
	})
}

//----------

func (sm *SM) AcceptId() bool {
	if !(sm.AcceptAny("_") ||
		sm.AcceptFn(unicode.IsLetter)) {
		return false
	}
	for sm.AcceptAny("_-") ||
		sm.AcceptLoopFn(unicode.IsLetter) ||
		sm.AcceptLoopFn(unicode.IsDigit) {
	}
	return true
}

func (sm *SM) AcceptInt() bool {
	p := sm.Pos
	ok := false
	_ = sm.AcceptAny("+-")
	if sm.AcceptLoopFn(unicode.IsDigit) {
		ok = true
	}
	if !ok {
		sm.Pos = p
		return false
	}
	return true
}

func (sm *SM) AcceptFloat() bool {
	p := sm.Pos
	ok := false
	_ = sm.AcceptAny("+-")
	if sm.AcceptLoopFn(unicode.IsDigit) {
		ok = true
	}
	if sm.AcceptAny(".") {
		ok = true
		_ = sm.AcceptLoopFn(unicode.IsDigit)
	}
	if sm.AcceptAny("eE") {
		ok = true
		_ = sm.AcceptAny("+-")
		_ = sm.AcceptLoopFn(unicode.IsDigit)
	}
	if !ok {
		sm.Pos = p
		return false
	}
	return true
}

//----------

func (sm *SM) AcceptQuoteLoop(quotes string, escapes string) bool {
	pos := sm.Pos
	ru := sm.Next()
	if sm.AcceptQuoteLoop2(ru, quotes, escapes) {
		return true
	}
	sm.Pos = pos
	return false
}

func (sm *SM) AcceptQuoteLoop2(quote rune, quotes string, escapes string) bool {
	if !strings.ContainsRune(quotes, quote) {
		return false
	}
	p := sm.Pos
	found := false
	_ = sm.AcceptLoopFn(func(ru rune) bool {
		if sm.AcceptEscape2(ru, escapes) {
			return true
		}
		if ru == quote {
			found = true
			return false
		}
		return true
	})
	if found {
		_ = sm.AcceptNRunes(1) // read end quote
		return true
	}
	sm.Pos = p
	return false
}

//----------

func (sm *SM) AcceptEscape2(ru rune, escapes string) bool {
	if escapes == "" {
		return false
	}
	if !strings.ContainsRune(escapes, ru) {
		return false
	}
	_ = sm.AcceptNRunes(1) // ignore result to allow EOS
	return true
}

//----------

func (sm *SM) AcceptToNewlineOrEnd() {
	_ = sm.AcceptLoopFn(func(ru rune) bool {
		return ru != '\n'
	})
}

//----------

func (sm *SM) Value() string {
	b, err := sm.r.ReadNSliceAt(sm.Start, sm.Pos-sm.Start)
	if err != nil {
		return ""
	}
	return string(b)
}

//----------

func (sm *SM) ValueInt() (int, error) {
	return strconv.Atoi(sm.Value())
}

func (sm *SM) ValueFloat() (float64, error) {
	return strconv.ParseFloat(sm.Value(), 64)
}

//----------

func (sm *SM) AcceptValueIntAdvance() (int, error) {
	if !sm.AcceptInt() {
		return 0, sm.Errorf("expecting int")
	}
	v, err := sm.ValueInt()
	if err != nil {
		return 0, err
	}
	sm.Advance()
	return v, nil
}

func (sm *SM) AcceptSpaceAdvance() error {
	if !sm.AcceptSpace() {
		return sm.Errorf("expecting space")
	}
	sm.Advance()
	return nil
}

//----------

//func (sm *SM) Item() *Item {
//	item := &Item{0, sm.Start, sm.Pos, sm.Input[sm.Start:sm.Pos]}
//	sm.Start = sm.Pos
//	return item
//}

func (sm *SM) Errorf(f string, args ...interface{}) error {
	// just n in each direction for error string
	pad := 30
	i0 := 0
	if sm.Pos-pad > i0 {
		i0 = sm.Pos - pad
	}
	i1 := sm.r.Len()
	if sm.Pos+pad < i0 {
		i0 = sm.Pos + pad
	}

	// context string with position indicator
	b1, err := sm.r.ReadNSliceAt(i0, i1)
	if err != nil {
		return err
	}
	s1 := string(b1)
	p := sm.Pos - i0
	ctx := s1[:p] + "***" + s1[p:]
	if i0 > 0 {
		ctx = "..." + ctx
	}
	if i1 < sm.r.Len() {
		ctx = ctx + "..."
	}

	msg := fmt.Sprintf(f, args...)
	return fmt.Errorf("%s: pos=%v [%v]", msg, sm.Pos, ctx)
}
