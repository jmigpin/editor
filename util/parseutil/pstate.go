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

// from provided, to current position
func (ps *PState) BytesFrom(from int) []byte {
	return ps.Src[from:ps.Pos]
}

//----------

//godebug:annotateoff
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
func (ps *PState) PeekRune() (rune, error) {
	pos0 := ps.Pos
	ru, err := ps.ReadRune()
	ps.Pos = pos0
	return ru, err
}

//----------

func (ps *PState) MatchRune(ru rune) error {
	ps2 := ps.Copy()
	ru2, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if ru2 != ru {
		return NoMatchErr
	}
	ps.Set(ps2)
	return nil
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
			return NoMatchErr
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
	return NoMatchErr
}

//----------

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
	return NoMatchErr
}
func (ps *PState) MatchRunesOrNeg(rs []rune) error { // negation
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if !ContainsRune(rs, ru) {
		ps.Set(ps2)
		return nil
	}
	return NoMatchErr
}

//----------

func (ps *PState) MatchRuneRanges(rrs RuneRanges) error {
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if rrs.HasRune(ru) {
		ps.Set(ps2)
		return nil
	}
	return NoMatchErr
}
func (ps *PState) MatchRuneRangesNeg(rrs RuneRanges) error { // negation
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if !rrs.HasRune(ru) {
		ps.Set(ps2)
		return nil
	}
	return NoMatchErr
}

//----------

func (ps *PState) MatchRunesAndRuneRanges(rs []rune, rrs RuneRanges) error { // negation
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if ContainsRune(rs, ru) || rrs.HasRune(ru) {
		ps.Set(ps2)
		return nil
	}
	return NoMatchErr
}
func (ps *PState) MatchRunesAndRuneRangesNeg(rs []rune, rrs RuneRanges) error { // negation
	ps2 := ps.Copy()
	ru, err := ps2.ReadRune()
	if err != nil {
		return err
	}
	if !ContainsRune(rs, ru) && !rrs.HasRune(ru) {
		ps.Set(ps2)
		return nil
	}
	return NoMatchErr
}

//----------

func (ps *PState) MatchString(s string) error {
	return ps.MatchRunesAnd([]rune(s)) // TODO: inneficient, better to use func with string that might need to test just the first rune and fail
}

//----------

func (ps *PState) MatchEof() error {
	ps2 := ps.Copy()
	_, err := ps2.ReadRune()
	if errors.Is(err, io.EOF) {
		ps.Set(ps2)
		return nil
	}
	return NoMatchErr
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

func (ps *PState) QuotedString() error {
	return ps.QuotedString2('\\', 3000, 10)
}

// allows escaped runes (if esc!=0)
func (ps *PState) QuotedString2(esc rune, maxLen1, maxLen2 int) error {
	q := "\"" // doublequote: fail on newline, eof doesn't close
	if err := ps.StringSection(q, q, esc, true, maxLen1, false); err == nil {
		return nil
	}
	q = "'" // singlequote: fail on newline, eof doesn't close (usually a smaller maxlen)
	if err := ps.StringSection(q, q, esc, true, maxLen2, false); err == nil {
		return nil
	}
	q = "`" // backquote: can have newline, eof doesn't close
	if err := ps.StringSection(q, q, esc, false, maxLen1, false); err == nil {
		return nil
	}
	return fmt.Errorf("not a quoted string")
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

func (ps *PState) OneOrMoreFn(fn func(rune) bool) error {
	for i := 0; ; i++ {
		ps2 := ps.Copy()
		ru, err := ps2.ReadRune()
		if err != nil {
			return err
		}
		if !fn(ru) {
			return NoMatchErr
		}
		ps.Set(ps2)
	}
}

//----------

func (ps *PState) Int() error {
	if err := ps.MatchRunesOr([]rune("+-")); err != nil {
		return err
	}
	return ps.OneOrMoreFn(unicode.IsDigit)
}

//----------
//----------
//----------

// parse node
type PNode interface {
	Pos() int
	End() int
}

//----------

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

//----------
//----------
//----------

// basic parse node implementation
type BasicPNode struct {
	pos int // can have pos>end when in reverse
	end int
}

func (n *BasicPNode) Pos() int {
	return n.pos
}
func (n *BasicPNode) End() int {
	return n.end
}
func (n *BasicPNode) SetPos(pos, end int) {
	n.pos = pos
	n.end = end
}
func (n *BasicPNode) PosEmpty() bool {
	return n.pos == n.end
}
func (n *BasicPNode) SrcString(src []byte) string {
	return string(src[n.pos:n.end])
}

//----------
//----------
//----------

type RuneRange [2]rune // assume [0]<[1]

func (rr RuneRange) HasRune(ru rune) bool {
	return ru >= rr[0] && ru <= rr[1]
}
func (rr RuneRange) IntersectsRange(rr2 RuneRange) bool {
	noIntersection := rr2[1] <= rr[0] || rr2[0] > rr[1]
	return !noIntersection
}
func (rr RuneRange) String() string {
	return fmt.Sprintf("%q-%q", rr[0], rr[1])
}

//----------

type RuneRanges []RuneRange

func (rrs RuneRanges) HasRune(ru rune) bool {
	for _, rr := range rrs {
		if rr.HasRune(ru) {
			return true
		}
	}
	return false
}

//----------
//----------
//----------

var NoMatchErr = errors.New("no match")

//var noParseErr = errors.New("no match") // TODO: indicates that the rule is not parseable, meaning the error is not deep enough to warrant a stop

//----------
//----------
//----------

//type matcher struct {
//	ps *PState
//}

//func (m *matcher) And(fns ...func() (any, error)) (any, error) {
//	index := func(i int) int { return i }
//	if m.ps.Reverse {
//		index = func(i int) int { return len(fns) - 1 - i }
//	}
//	for i := 0; i < len(fns); i++ {
//		fn := fns[index(i)]
//		if v, err := fn(); err != nil {
//			return nil, err
//		}
//	}
//	return true
//}

//func (m *matcher) Int() error {
//	for{
//			_ = m.Any("+-")
//			return true
//		},
//		func() bool {
//			return m.FnLoop(unicode.IsDigit)
//		})
//}

//func ParseAnd(ps *PState) error {

//}
