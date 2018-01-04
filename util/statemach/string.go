package statemach

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const EOF = -1

type String struct {
	Start int
	Pos   int
	Input string
}

func NewString(input string) *String {
	return &String{
		Input: input,
	}
}

func (sm *String) Next() rune {
	if sm.Pos >= len(sm.Input) {
		return EOF
	}
	ru, w := utf8.DecodeRuneInString(sm.Input[sm.Pos:])
	sm.Pos += w
	return ru
}

func (sm *String) Peek() rune {
	p := sm.Pos
	u := sm.Next()
	sm.Pos = p
	return u
}

func (sm *String) Accept(valid string) bool {
	pos := sm.Pos
	if strings.ContainsRune(valid, sm.Next()) {
		return true
	}
	sm.Pos = pos
	return false
}

func (sm *String) AcceptNeg(invalid string) bool {
	pos := sm.Pos
	if !strings.ContainsRune(invalid, sm.Next()) {
		return true
	}
	sm.Pos = pos
	return false
}

func (sm *String) AcceptSequence(s string) bool {
	p := sm.Pos
	for _, ru := range s {
		if ru != sm.Next() {
			sm.Pos = p
			return false
		}
	}
	return true
}

func (sm *String) AcceptFn(fn func(rune) bool) bool {
	p := sm.Pos
	if fn(sm.Next()) {
		return true
	}
	sm.Pos = p
	return false
}

func (sm *String) AcceptLoop(valid string) bool {
	v := false
	for sm.Accept(valid) {
		v = true
	}
	return v
}

func (sm *String) AcceptLoopFn(fn func(rune) bool) bool {
	v := false
	for sm.AcceptFn(fn) {
		v = true
	}
	return v
}

func (sm *String) AcceptSpace() bool {
	return sm.AcceptLoopFn(unicode.IsSpace)
}

func (sm *String) AcceptId() bool {
	if !(sm.Accept("_") ||
		sm.AcceptFn(unicode.IsLetter)) {
		return false
	}
	for sm.Accept("_-") ||
		sm.AcceptLoopFn(unicode.IsLetter) ||
		sm.AcceptLoopFn(unicode.IsDigit) {
	}
	return true
}

func (sm *String) AcceptInt() bool {
	p := sm.Pos
	ok := false
	_ = sm.Accept("+-")
	if sm.AcceptLoopFn(unicode.IsDigit) {
		ok = true
	}
	if !ok {
		sm.Pos = p
		return false
	}
	return true
}

func (sm *String) AcceptFloat() bool {
	p := sm.Pos
	ok := false
	_ = sm.Accept("+-")
	if sm.AcceptLoopFn(unicode.IsDigit) {
		ok = true
	}
	if sm.Accept(".") {
		ok = true
		_ = sm.AcceptLoopFn(unicode.IsDigit)
	}
	if sm.Accept("eE") {
		ok = true
		_ = sm.Accept("+-")
		_ = sm.AcceptLoopFn(unicode.IsDigit)
	}
	if !ok {
		sm.Pos = p
		return false
	}
	return true
}

func (sm *String) AcceptToNewlineOrEOF() {
	for {
		ru := sm.Next()
		if ru == '\n' || ru == EOF {
			break
		}
	}
}

func (sm *String) AcceptN(n int) bool {
	if sm.Pos+n > len(sm.Input) {
		return false
	}
	sm.Pos += n
	return true
}

func (sm *String) Advance() {
	sm.Start = sm.Pos
}

func (sm *String) Value() string {
	return sm.Input[sm.Start:sm.Pos]
}

func (sm *String) ValueInt() (int, error) {
	return strconv.Atoi(sm.Value())
}

func (sm *String) ValueFloat() (float64, error) {
	return strconv.ParseFloat(sm.Value(), 64)
}

func (sm *String) AcceptValueIntAdvance() (int, error) {
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

func (sm *String) AcceptSpaceAdvance() error {
	if !sm.AcceptSpace() {
		return sm.Errorf("expecting space")
	}
	sm.Advance()
	return nil
}

//func (sm *String) Item() *Item {
//	item := &Item{0, sm.Start, sm.Pos, sm.Input[sm.Start:sm.Pos]}
//	sm.Start = sm.Pos
//	return item
//}

func (sm *String) Errorf(f string, args ...interface{}) error {
	// just n in each direction for error string
	pad := 30
	i0 := 0
	if sm.Pos-pad > i0 {
		i0 = sm.Pos - pad
	}
	i1 := len(sm.Input)
	if sm.Pos+pad < i0 {
		i0 = sm.Pos + pad
	}

	// context string with position indicator
	ctx := sm.Input[i0:sm.Pos] + "***" + sm.Input[sm.Pos:i1]
	if i0 > 0 {
		ctx = "..." + ctx
	}
	if i1 < len(sm.Input) {
		ctx = ctx + "..."
	}

	msg := fmt.Sprintf(f, args...)
	return fmt.Errorf("%s: pos=%v [%v]", msg, sm.Pos, ctx)
}
