package parseutil

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/util/iout/iorw"
)

// WARNING: user functions are responsible for rewinding on fail; check sc.Or(...), sc.Optional(...), etc.
type Scanner struct {
	Pos        int
	Reverse    bool // read direction
	ErrPadSize int

	q  []Node
	rd iorw.ReaderAt

	DebugRead bool
	debugPos  int

	stopErr    error
	scanErr    error
	scanErrPos int
}

func NewScanner() *Scanner {
	sc := &Scanner{}
	sc.ErrPadSize = 40      // full size
	sc.q = make([]Node, 16) // alloc here for performance
	return sc
}
func NewScannerFromReader(rd iorw.ReaderAt) *Scanner {
	sc := NewScanner()
	sc.SetReader(rd)
	return sc
}
func NewScannerFromBytes(src []byte) *Scanner {
	sc := NewScanner()
	sc.SetBytes(src)
	return sc
}
func NewScannerFromString(src string) *Scanner {
	return NewScannerFromBytes([]byte(src))
}

//----------

func (sc *Scanner) SetBytes(b []byte) {
	sc.SetReader(iorw.NewBytesReadWriterAt(b))
}
func (sc *Scanner) SetReader(rd iorw.ReaderAt) {
	sc.rd = rd
	sc.Reset()
}
func (sc *Scanner) Reset() {
	sc.ResetPos()
	sc.ResetErrors()
	sc.ResetNodeQ()
}
func (sc *Scanner) ResetPos() {
	sc.Pos = 0
	sc.debugPos = -1
}
func (sc *Scanner) ResetErrors() {
	sc.stopErr = nil
	sc.scanErr = nil
	sc.scanErrPos = 0
}

//----------

// usage: a := sc.Pointer(); sc.And(a.Ptr); ... ; a.SetPtr(...)
func (sc *Scanner) Pointer() ScanFuncPtr {
	return ScanFuncPtr{}
}

//----------

func (sc *Scanner) ReadByte() (byte, error) {
	if sc.stopErr != nil {
		return 0, sc.stopErr
	}
	if sc.DebugRead {
		sc.printDebugPosCtx()
	}

	if sc.Reverse {
		b, err := sc.rd.ReadFastAt(sc.Pos-1, sc.Pos)
		if err != nil {
			return 0, err
		}
		sc.Pos -= 1
		return b[0], nil
	}

	b, err := sc.rd.ReadFastAt(sc.Pos, sc.Pos+1)
	if err != nil {
		return 0, err
	}
	sc.Pos += 1
	return b[0], nil
}

func (sc *Scanner) ReadRune() (rune, error) {
	if sc.stopErr != nil {
		return 0, sc.stopErr
	}
	if sc.DebugRead {
		sc.printDebugPosCtx()
	}

	if sc.Reverse {
		ru, w, err := iorw.ReadLastRuneAt(sc.rd, sc.Pos)
		if err != nil {
			return 0, err
		}
		sc.Pos -= w
		return ru, nil
	}

	ru, w, err := iorw.ReadRuneAt(sc.rd, sc.Pos)
	if err != nil {
		return 0, err
	}
	sc.Pos += w
	return ru, nil
}

//----------

func (sc *Scanner) PeekRune() (rune, error) {
	pos0 := sc.Pos
	ru, err := sc.ReadRune()
	sc.Pos = pos0 // always rewind
	return ru, err
}

//----------

func (sc *Scanner) RewindOnFalse(fn ScanFunc) bool {
	return sc.noNodesOnFalse(func() bool {
		pos0 := sc.Pos
		if !fn() {
			sc.Pos = pos0 // rewind
			return false
		}
		return true
	})

	//// performance
	//pos0 := sc.Pos
	//q0 := sc.q
	//sc.q = sc.q[len(q0):]
	//if !fn() {
	//	sc.Pos = pos0 // rewind
	//	sc.q = q0     // reset
	//	return false
	//}
	//sc.q = append(q0, sc.q...) // add to original
	//return true
}
func (sc *Scanner) Rewind(fn ScanFunc) bool {
	pos0 := sc.Pos
	r := fn()
	sc.Pos = pos0 // always rewind
	return r
}
func (sc *Scanner) FnOnTrue(fn, trueFn ScanFunc) bool {
	// not equivalent to And(...) since it works the same in reverse

	// must rewind here on false because trueFn might fail
	return sc.RewindOnFalse(func() bool {
		if !fn() {
			return false
		}
		return trueFn()
	})
}
func (sc *Scanner) PosFnOnTrue(fn ScanFunc, trueFn PosFunc) bool {
	start := sc.Pos
	return sc.FnOnTrue(
		fn,
		func() bool {
			return trueFn(start, sc.Pos)
		},
	)
}
func (sc *Scanner) UseReverse(reverse bool, fn ScanFunc) bool {
	t := sc.Reverse
	sc.Reverse = reverse
	defer func() { sc.Reverse = t }() // restore
	return fn()
}
func (sc *Scanner) ExpandReverse(fn ScanFunc) bool {
	return sc.noNodesOnFalse(sc.UseReverseF(true, func() bool {
		ok := fn()
		sc.ResetNodeQ()
		return ok
	}))
}

//----------

func (sc *Scanner) And(fns ...ScanFunc) bool {
	// NOTE: pitfall: not able to accept a single space since it will match the first optional; it would work if the optional is in the 2nd line
	// 	sc.And(
	// 		sc.OptionalF(sc.RuneF(' ')),
	// 		sc.RuneF(' '),
	// 	)

	// must rewind on false since some fns might have suceeded
	return sc.RewindOnFalse(func() bool {
		if sc.Reverse {
			for i := len(fns) - 1; i >= 0; i-- {
				fn := fns[i]
				if !fn() {
					return false
				}
			}
			return true
		}
		for i := 0; i < len(fns); i++ {
			fn := fns[i]
			if !fn() {
				return false
			}
		}
		return true
	})
}
func (sc *Scanner) Or(fns ...ScanFunc) bool {
	// same code if in reverse mode

	for i := 0; i < len(fns); i++ {
		fn := fns[i]
		if fn() {
			return true
		}
	}
	return false
}
func (sc *Scanner) Optional(fn ScanFunc) bool {
	_ = fn()
	return true
}

// Usage example when building slices
// 	func() bool {
// 		res:=[]int{}
// 		return sc.FnOnTrue(
// 			sc.LoopF(func() bool{ res=append(res, ...) ...}),
// 			func()bool{
// 				sc.PushBackNode(res) // only on true
// 			},
// 		)
// 	}
func (sc *Scanner) Loop(fn ScanFunc) bool {
	for first := true; ; first = false {
		if !fn() {
			return !first
		}
	}
}
func (sc *Scanner) LoopSep(fn, sepFn ScanFunc) bool {
	for first := true; ; first = false {
		if !fn() {
			return !first
		}
		if !sepFn() {
			// have at least one result
			// allows ending with a separator
			return true
		}
	}
}
func (sc *Scanner) LoopRuneFn(fn RuneFunc) bool {
	//return sc.Loop(func() bool {
	//	return sc.RuneFn(fn)
	//})

	// performance
	for first := true; ; first = false {
		pos0 := sc.Pos
		ru, err := sc.ReadRune()
		if err != nil || !fn(ru) {
			sc.Pos = pos0 // rewind last rune
			return !first
		}
	}
}
func (sc *Scanner) LoopUntil(maxN int, fn, untilFn ScanFunc) bool {
	return sc.RewindOnFalse(func() bool {
		for first := true; ; first = false {
			if untilFn() {
				return true
			}
			maxN--
			if maxN < 0 {
				return false
			}
			if !fn() {
				return !first
			}
		}
	})
}

//----------

func (sc *Scanner) ByteFn(fn ByteFunc) bool {
	// fn can't rewind to the location before the readbyte, do it here
	return sc.RewindOnFalse(func() bool {
		b, err := sc.ReadByte()
		if err != nil {
			// error info lost
			return false
		}
		return fn(b)
	})
}
func (sc *Scanner) RuneFn(fn RuneFunc) bool {
	// fn can't rewind to the location before the readrune, do it here
	return sc.RewindOnFalse(func() bool {
		ru, err := sc.ReadRune()
		if err != nil {
			// error info lost
			return false
		}
		return fn(ru)
	})
}
func (sc *Scanner) RuneNotFn(fn RuneFunc) bool {
	return sc.RuneFn(func(ru rune) bool {
		return !fn(ru)
	})
}
func (sc *Scanner) Byte(b byte) bool {
	return sc.RewindOnFalse(func() bool {
		u, err := sc.ReadByte()
		if err != nil {
			// error info lost
			return false
		}
		return u == b
	})
}
func (sc *Scanner) ByteRange(b1, b2 byte) bool {
	return sc.ByteFn(func(b byte) bool {
		return b >= b1 && b <= b2
	})
}
func (sc *Scanner) Rune(ru rune) bool {
	return sc.RuneFn(func(ru2 rune) bool {
		return ru2 == ru
	})
}
func (sc *Scanner) RuneRange(ru1, ru2 rune) bool {
	return sc.RuneFn(func(ru rune) bool {
		return ru >= ru1 && ru <= ru2
	})
}
func (sc *Scanner) RunePeek(ru rune) bool {
	ru2, err := sc.PeekRune()
	return err == nil && ru == ru2
}
func (sc *Scanner) Any(valid string) bool {
	return sc.RuneFn(func(ru rune) bool {
		return strings.ContainsRune(valid, ru)
	})
}
func (sc *Scanner) Except(invalid string) bool {
	return sc.RuneFn(func(ru rune) bool {
		return !strings.ContainsRune(invalid, ru)
	})
}
func (sc *Scanner) Sequence(s string) bool {
	return sc.RewindOnFalse(func() bool {
		fn := func(ru rune) bool {
			ru2, err := sc.ReadRune()
			if err != nil || ru != ru2 {
				// error info lost
				return false
			}
			return true
		}
		if sc.Reverse {
			w := []rune(s)
			for i := len(w) - 1; i >= 0; i-- {
				if !fn(w[i]) {
					return false
				}
			}
		} else {
			for _, ru := range s {
				if !fn(ru) {
					return false
				}
			}
		}
		return true
	})
}

func (sc *Scanner) SequenceExpand(s string) bool {
	return sc.RewindOnFalse(func() bool {
		w := []rune(s)
		if sc.Reverse {
			for k := len(w) - 1; k >= 0; k-- {
				if sc.Sequence(s) {
					return true
				}
				sc.Pos += len(string(w[k]))
			}
		} else {
			for k := 0; k < len(w); k++ {
				if sc.Sequence(s) {
					return true
				}
				sc.Pos -= len(string(w[k]))
			}
		}
		return false
	})
}
func (sc *Scanner) NRunes(n int) bool {
	return sc.RewindOnFalse(func() bool {
		for c := 0; c < n; c++ {
			_, err := sc.ReadRune()
			if err != nil {
				// error info lost
				return false
			}
		}
		return true
	})
}

//----------

func (sc *Scanner) End() bool {
	if sc.stopErr != nil {
		return false
	}
	_, err := sc.PeekRune()
	return err != nil // testing for io.EOF could fail in limited readers
}
func (sc *Scanner) Spaces() bool {
	return sc.LoopRuneFn(unicode.IsSpace)
}
func (sc *Scanner) SpacesExceptNewline() bool {
	return sc.LoopRuneFn(func(ru rune) bool {
		return ru != '\n' && unicode.IsSpace(ru)
	})
}
func (sc *Scanner) SpacesExceptNewline2(escape byte) bool {
	return sc.Loop(sc.OrF(
		// allow escaping only spaces
		sc.EscapedRuneFnF(escape, func(ru rune) bool {
			return unicode.IsSpace(ru)
		}),
		sc.RuneFnF(func(ru rune) bool {
			return ru != '\n' && unicode.IsSpace(ru)
		}),
	))
}
func (sc *Scanner) LoopAnyExceptSpaces(escape byte) bool {
	return sc.Loop(sc.OrF(
		sc.EscapedRuneF(escape),
		sc.RuneFnF(func(ru rune) bool {
			return !unicode.IsSpace(ru)
		}),
	))
}
func (sc *Scanner) EscapedRune(escape byte) bool {
	return sc.RewindOnFalse(sc.AndF(
		func() bool { return sc.validateNearEscapes(escape) },
		sc.ByteF(escape),
		sc.NRunesF(1),
	))
}
func (sc *Scanner) EscapedRuneFn(escape byte, fn RuneFunc) bool {
	return sc.RewindOnFalse(sc.AndF(
		func() bool { return sc.validateNearEscapes(escape) },
		sc.ByteF(escape),
		sc.RuneFnF(fn),
	))
}
func (sc *Scanner) validateNearEscapes(escape byte) bool {
	if sc.Reverse {
		pos0 := sc.Pos
		// count number of escapes
		c := 0
		for ; ; c++ {
			if !sc.Byte(escape) {
				break
			}
		}
		if c%2 != 0 {
			// the escape is itself escaped, so not an escape
			return false
		}
		sc.Pos = pos0 // rewind before accepting
		return true   // accept
	}
	// accept if not in reverse
	return true
}
func (sc *Scanner) Section(open, close string, escape byte, failOnNewline bool, maxLen int, eofClose bool) bool {
	return sc.RewindOnFalse(func() bool {
		start := sc.Pos
		if !sc.Sequence(open) {
			return false
		}
		for {
			if escape != 0 && sc.EscapedRune(escape) {
				continue
			}
			if sc.Sequence(close) {
				return true
			}
			ru, err := sc.ReadRune() // consume rune
			if err != nil {
				// extension: stop on eof
				if err == io.EOF {
					return eofClose
				}
				// error info lost
				return false
			}
			// extension: newline
			if ru == '\n' && failOnNewline {
				return false
			}
			// extension: stop on maxlength
			if maxLen > 0 {
				d := sc.Pos - start
				if d < 0 {
					d = -d
				}
				if d >= maxLen {
					return false
				}
			}
		}
	})
}

//----------

func (sc *Scanner) LoopAnyToNewlineExcludeOrEnd() bool {
	_ = sc.LoopRuneFn(func(ru rune) bool {
		return ru != '\n'
	})
	return true // allow empty
}
func (sc *Scanner) LoopAnyToNewlineIncludeOrEnd() bool {
	for {
		ru, err := sc.ReadRune()
		if err != nil {
			// error info lost
			return true // also allows empty
		}
		if ru == '\n' {
			return true
		}
	}
}
func (sc *Scanner) LoopAnyToNewlineIncludeOrEndWithEscapes(escape byte) bool {
	esc := rune(escape)
	for {
		ru, err := sc.ReadRune()
		if err != nil {
			// error info lost
			return true // also allows empty
		}
		switch ru {
		case esc:
			_ = sc.NRunes(1) // allows escaping a newline
		case '\n':
			return true
		}
	}
}

//----------

func (sc *Scanner) QuotedString(quote rune, escape byte, failOnNewline bool, maxLen int) bool {
	q := string(quote)
	return sc.Section(q, q, escape, failOnNewline, maxLen, false)
}
func (sc *Scanner) QuotedString2(validQuotes string, escape byte, failOnNewline bool, maxLen int) bool {
	ru, err := sc.PeekRune()
	return err == nil &&
		strings.ContainsRune(validQuotes, ru) &&
		sc.QuotedString(ru, escape, failOnNewline, maxLen)
}
func (sc *Scanner) DoubleQuotedString(escape byte) bool {
	return sc.QuotedString('"', escape, true, 0)
}
func (sc *Scanner) SingleQuotedString(escape byte) bool {
	return sc.QuotedString('\'', escape, true, 0)
}
func (sc *Scanner) MultiLineComment() bool {
	return sc.Section("/*", "*/", 0, false, 0, false)
}
func (sc *Scanner) LineComment() bool {
	return sc.Section("//", "\n", 0, true, 0, true)
}
func (sc *Scanner) GoQuotes(escape byte, maxLen, maxLenSingleQuote int) bool {
	return sc.Or(
		sc.QuotedStringF('"', escape, true, maxLen),
		sc.QuotedStringF('`', escape, false, maxLen),
		sc.QuotedStringF('\'', escape, true, maxLenSingleQuote),
	)
}

//----------

func (sc *Scanner) Id1() bool {
	// TODO: old code included hiffen "-"
	// TODO: svgutil/astparser needs hiffen?

	//	if sc.Reverse {
	//		// unable to detect start is not digit in reverse
	//		// unable to backtrack the loop
	//		panic("can't parse in reverse")
	//	}

	//	return sc.And( // attempt at allowing reverse
	//		sc.OrF(
	//			sc.AnyF("_"),
	//			sc.RuneFnF(unicode.IsLetter),
	//		),
	//		sc.OptionalF(
	//			sc.LoopF(sc.OrF(
	//				sc.AnyF("_"),
	//				sc.RuneFnF(unicode.IsLetter),
	//				sc.Digits,
	//			)),
	//		),
	//	)

	// TODO: make check at the end if the first rune is not a digit
	if sc.Reverse {
		//return false
		panic("todo: reverse id1")
	}

	c := 0
	return sc.LoopRuneFn(func(ru rune) bool {
		c++
		return ru == '_' ||
			unicode.IsLetter(ru) ||
			(c >= 2 && unicode.IsDigit(ru))
	})
}

func (sc *Scanner) Digits() bool {
	return sc.LoopRuneFn(unicode.IsDigit)
}
func (sc *Scanner) Hexadecimal() bool {
	return sc.LoopRuneFn(func(ru rune) bool {
		return unicode.IsDigit(ru) ||
			// a-f and A-F
			(ru >= 'a' && ru <= 'f') ||
			(ru >= 'A' && ru <= 'F')
	})
}

func (sc *Scanner) Integer() bool {
	return sc.And( // allow reverse
		sc.OptionalF(sc.AnyF("+-")),
		sc.Digits,
	)
}

func (sc *Scanner) Float() bool {
	fraction := sc.AndF(
		sc.RuneF('.'),
		sc.LoopRuneFnF(unicode.IsDigit),
	)

	return sc.And( // allows reverse attempt
		sc.OptionalF(sc.AnyF("+-")),
		sc.OrF(
			fraction,
			sc.AndF(
				sc.RuneF('0'),
				fraction,
			),
			sc.AndF(
				sc.RuneRangeF('1', '9'),
				sc.OptionalF(sc.LoopRuneFnF(unicode.IsDigit)),
				sc.OptionalF(fraction),
			),
		),
		sc.OptionalF(sc.AndF(
			sc.AnyF("eE"),
			sc.OptionalF(sc.AnyF("+-")),
			sc.LoopRuneFnF(unicode.IsDigit),
		)),
	)
}

//----------

func (sc *Scanner) BuildNode(fn, trueFn ScanFunc) bool {
	return sc.FnOnTrue(fn, trueFn)
}
func (sc *Scanner) BuildPosNode(fn ScanFunc, trueFn PosFunc) bool {
	return sc.PosFnOnTrue(fn, trueFn)
}

// Ensures single node on true
func (sc *Scanner) SingleNode(fn ScanFunc) bool {
	return sc.BuildNode(
		fn,
		func() bool {
			l := sc.NodeQLen()
			switch l {
			case 0:
				// useful for optionals that didn't build a node
				sc.PushBackNode(nil)
			case 1: // nothing todo
			default: // group into single node on >=2
				w := []Node{}
				for i := 0; i < l; i++ {
					w = append(w, sc.PopFrontNode())
				}
				sc.PushBackNode(w)
			}
			return true
		},
	)
}
func (sc *Scanner) BytesNode(fn ScanFunc) bool {
	v, ok := sc.BytesValue(fn)
	if !ok {
		return false
	}
	// make a copy (allows src to be changed without affecting nodes)
	v2 := make([]byte, len(v))
	copy(v2, v)

	sc.PushBackNode(v2)
	return true
}
func (sc *Scanner) StringNode(fn ScanFunc) bool {
	v, ok := sc.StringValue(fn)
	if !ok {
		return false
	}
	sc.PushBackNode(v)
	return true
}
func (sc *Scanner) StringUnquoteNode(fn ScanFunc) bool {
	v, ok := sc.StringUnquoteValue(fn)
	if !ok {
		return false
	}
	sc.PushBackNode(v)
	return true
}
func (sc *Scanner) IntNode(fn ScanFunc) bool {
	v, ok := sc.IntValue(fn)
	if !ok {
		return false
	}
	sc.PushBackNode(v)
	return true
}
func (sc *Scanner) Float64Node(fn ScanFunc) bool {
	v, ok := sc.Float64Value(fn)
	if !ok {
		return false
	}
	sc.PushBackNode(v)
	return true
}

//----------

func (sc *Scanner) BytesValue(fn ScanFunc) ([]byte, bool) {
	start := sc.Pos
	if !fn() {
		return nil, false
	}
	end := sc.Pos
	if sc.Reverse {
		start, end = end, start
	}
	b, err := sc.rd.ReadFastAt(start, end)
	if err != nil {
		// error info lost
		return nil, false
	}
	return b, true
}
func (sc *Scanner) StringValue(fn ScanFunc) (string, bool) {
	v, ok := sc.BytesValue(fn)
	return string(v), ok
}
func (sc *Scanner) StringValueFn(fn ScanFunc, trueFn func(string) bool) bool {
	// must rewind in case truefn fails
	return sc.RewindOnFalse(func() bool {
		v, ok := sc.StringValue(fn)
		return ok && trueFn(v)
	})
}
func (sc *Scanner) StringUnquoteValue(fn ScanFunc) (string, bool) {
	b, ok := sc.BytesValue(fn)
	if !ok {
		return "", false
	}

	//// strict: fail if it has no quotes
	//s, err := strconv.Unquote(string(b))
	//if err != nil {
	//	// error info lost
	//	return "", false
	//}

	// best effort to unquote
	s := string(b)
	s2, err := strconv.Unquote(s)
	if err == nil {
		s = s2
	}
	return s, true
}
func (sc *Scanner) IntValue(fn ScanFunc) (int, bool) {
	b, ok := sc.BytesValue(fn)
	if !ok {
		return 0, false
	}
	v, err := strconv.Atoi(string(b))
	if err != nil {
		// error info lost
		return 0, false
	}
	return v, true
}
func (sc *Scanner) Float64Value(fn ScanFunc) (float64, bool) {
	b, ok := sc.BytesValue(fn)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		// error info lost
		return 0, false
	}
	return v, true
}

//----------

func (sc *Scanner) Result(mainFn ScanFunc) (Node, error) {
	ok := mainFn()
	if !ok {
		if sc.stopErr != nil {
			return nil, sc.stopErr
		}
		if sc.scanErr != nil {
			return nil, sc.scanErr
		}
		return nil, fmt.Errorf("scan fail")
	}
	// setup result
	switch len(sc.q) {
	case 0:
		return nil, nil
	case 1:
		return sc.q[0], nil
	default:
		return sc.q, nil
	}
}

//----------

// push/pop node q
// ex: ensures that some calls to sc.And(...) that succeed won't leave nodes in the Q if it fails in the end
func (sc *Scanner) noNodesOnFalse(fn ScanFunc) bool {
	q0 := sc.q // keep
	//sc.q = []Node{} // reset
	sc.q = sc.q[len(q0):] // reset (performance: avoid alloc)
	if !fn() {
		sc.q = q0 // restore, discards pushed nodes
		return false
	}
	sc.q = append(q0, sc.q...) // add to original
	return true
}

func (sc *Scanner) PopFrontNode() Node {
	// TODO: if reverse, pop in reverse to match code order? what about optionals?

	l := len(sc.q)
	if l == 0 {
		return nil
	}
	front := sc.q[0]
	sc.q = sc.q[1:] // pop front
	return front
}
func (sc *Scanner) PushBackNode(node Node) {
	sc.q = append(sc.q, node)
}
func (sc *Scanner) NodeQLen() int {
	return len(sc.q)
}
func (sc *Scanner) ResetNodeQ() {
	//sc.q = nil
	sc.q = sc.q[:0] // performance
}

//----------

// OTHERS: Printlnf

//func (sc *Scanner4) DebugNodeQ() bool {
//	fmt.Printf("nodeq:\n")
//	for _, v := range sc.q {
//		fmt.Printf("\t%#v\n", v)
//	}
//	return true
//}

//----------
//----------
//----------

func (sc *Scanner) RewindOnFalseF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.RewindOnFalse(fn) }
}
func (sc *Scanner) RewindF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.Rewind(fn) }
}
func (sc *Scanner) FnOnTrueF(fn, trueFn ScanFunc) ScanFunc {
	return func() bool { return sc.FnOnTrue(fn, trueFn) }
}
func (sc *Scanner) PosFnOnTrueF(fn ScanFunc, trueFn PosFunc) ScanFunc {
	return func() bool { return sc.PosFnOnTrue(fn, trueFn) }
}
func (sc *Scanner) UseReverseF(reverse bool, fn ScanFunc) ScanFunc {
	return func() bool { return sc.UseReverse(reverse, fn) }
}
func (sc *Scanner) ExpandReverseF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.ExpandReverse(fn) }
}

//----------

// variants ending in "*F" that return a function

func (sc *Scanner) AndF(fns ...ScanFunc) ScanFunc {
	return func() bool { return sc.And(fns...) }
}
func (sc *Scanner) OrF(fns ...ScanFunc) ScanFunc {
	return func() bool { return sc.Or(fns...) }
}
func (sc *Scanner) OptionalF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.Optional(fn) }
}
func (sc *Scanner) LoopF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.Loop(fn) }
}
func (sc *Scanner) LoopSepF(fn, sepFn ScanFunc) ScanFunc {
	return func() bool { return sc.LoopSep(fn, sepFn) }
}
func (sc *Scanner) LoopRuneFnF(fn RuneFunc) ScanFunc {
	return func() bool { return sc.LoopRuneFn(fn) }
}
func (sc *Scanner) LoopUntilF(maxN int, fn, untilFn ScanFunc) ScanFunc {
	return func() bool { return sc.LoopUntil(maxN, fn, untilFn) }
}

//----------

func (sc *Scanner) ByteFnF(fn ByteFunc) ScanFunc {
	return func() bool { return sc.ByteFn(fn) }
}
func (sc *Scanner) RuneFnF(fn RuneFunc) ScanFunc {
	return func() bool { return sc.RuneFn(fn) }
}
func (sc *Scanner) RuneNotFnF(fn RuneFunc) ScanFunc {
	return func() bool { return sc.RuneNotFn(fn) }
}
func (sc *Scanner) ByteF(b byte) ScanFunc {
	return func() bool { return sc.Byte(b) }
}
func (sc *Scanner) ByteRangeF(b1, b2 byte) ScanFunc {
	return func() bool { return sc.ByteRange(b1, b2) }
}
func (sc *Scanner) RuneF(ru rune) ScanFunc {
	return func() bool { return sc.Rune(ru) }
}
func (sc *Scanner) RuneRangeF(ru1, ru2 rune) ScanFunc {
	return func() bool { return sc.RuneRange(ru1, ru2) }
}
func (sc *Scanner) RunePeekF(ru rune) ScanFunc {
	return func() bool { return sc.RunePeek(ru) }
}
func (sc *Scanner) AnyF(valid string) ScanFunc {
	return func() bool { return sc.Any(valid) }
}
func (sc *Scanner) ExceptF(invalid string) ScanFunc {
	return func() bool { return sc.Except(invalid) }
}
func (sc *Scanner) SequenceF(seq string) ScanFunc {
	return func() bool { return sc.Sequence(seq) }
}
func (sc *Scanner) SequenceExpandF(seq string) ScanFunc {
	return func() bool { return sc.SequenceExpand(seq) }
}
func (sc *Scanner) NRunesF(n int) ScanFunc {
	return func() bool { return sc.NRunes(n) }
}

//----------

// NOTE: EofF() // has no args, not needed
// NOTE: SpacesF() // has no args, not needed
// NOTE: SpacesExceptNewlineF() // has no args, not needed

func (sc *Scanner) SpacesExceptNewline2F(escape byte) ScanFunc {
	return func() bool { return sc.SpacesExceptNewline2(escape) }
}
func (sc *Scanner) LoopAnyExceptSpacesF(escape byte) ScanFunc {
	return func() bool { return sc.LoopAnyExceptSpaces(escape) }
}
func (sc *Scanner) EscapedRuneF(escape byte) ScanFunc {
	return func() bool { return sc.EscapedRune(escape) }
}
func (sc *Scanner) EscapedRuneFnF(escape byte, fn RuneFunc) ScanFunc {
	return func() bool { return sc.EscapedRuneFn(escape, fn) }
}
func (sc *Scanner) SectionF(open, close string, escape byte, failOnNewline bool, maxLen int, eofClose bool) ScanFunc {
	return func() bool { return sc.Section(open, close, escape, failOnNewline, maxLen, eofClose) }
}

//----------

// NOTE: LoopAnyToNewlineExcludeOrEnd() // has no args, not needed
// NOTE: LoopAnyToNewlineIncludeOrEnd() // has no args, not needed

//----------

// TODO: other string funcs

func (sc *Scanner) QuotedStringF(quote rune, escape byte, failOnNewline bool, maxLen int) ScanFunc {
	return func() bool { return sc.QuotedString(quote, escape, failOnNewline, maxLen) }
}
func (sc *Scanner) QuotedString2F(validQuotes string, escape byte, failOnNewline bool, maxLen int) ScanFunc {
	return func() bool { return sc.QuotedString2(validQuotes, escape, failOnNewline, maxLen) }
}
func (sc *Scanner) DoubleQuotedStringF(escape byte) ScanFunc {
	return func() bool { return sc.DoubleQuotedString(escape) }
}

// TODO: other funcs

func (sc *Scanner) GoQuotesF(escape byte, maxLen, maxLenSingleQuote int) ScanFunc {
	return func() bool { return sc.GoQuotes(escape, maxLen, maxLenSingleQuote) }
}

//----------

// TODO: other utils funcs (id1, integer, float, ...)

//----------

func (sc *Scanner) BuildNodeF(fn, trueFn ScanFunc) ScanFunc {
	return func() bool { return sc.BuildNode(fn, trueFn) }
}
func (sc *Scanner) BuildPosNodeF(fn ScanFunc, trueFn PosFunc) ScanFunc {
	return func() bool { return sc.BuildPosNode(fn, trueFn) }
}
func (sc *Scanner) SingleNodeF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.SingleNode(fn) }
}
func (sc *Scanner) BytesNodeF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.BytesNode(fn) }
}
func (sc *Scanner) StringNodeF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.StringNode(fn) }
}
func (sc *Scanner) StringUnquoteNodeF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.StringUnquoteNode(fn) }
}
func (sc *Scanner) IntNodeF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.IntNode(fn) }
}
func (sc *Scanner) Float64NodeF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.Float64Node(fn) }
}

//----------

// TODO: values

func (sc *Scanner) StringValueFnF(fn ScanFunc, trueFn func(string) bool) ScanFunc {
	return func() bool { return sc.StringValueFn(fn, trueFn) }
}

//----------

// TODO: resultf

//----------

func (sc *Scanner) noNodesOnFalseF(fn ScanFunc) ScanFunc {
	return func() bool { return sc.noNodesOnFalse(fn) }
}

//----------

func (sc *Scanner) PrintlnF(v bool, s string) ScanFunc {
	return func() bool { fmt.Println(s); return v }
}
func (sc *Scanner) PrintPosF(fn ScanFunc) ScanFunc {
	return func() bool {
		pos0 := sc.Pos
		ok := fn()
		if ok {
			fmt.Printf("printpos: %v->%v\n", pos0, sc.Pos)
		}
		return ok
	}
}

// OTHERS: DebugNodeQ

//----------

func (sc *Scanner) CtxString() string {
	s, err := CtxString(sc.rd, sc.Pos, sc.ErrPadSize)
	if err != nil {
		return err.Error()
	}
	return s
}

func (sc *Scanner) printDebugPosCtx() {
	if sc.Pos > sc.debugPos {
		sc.debugPos = sc.Pos
		rev := ""
		if sc.Reverse {
			rev = " rev"
		}
		fmt.Printf("dbg%s: %v: %q\n", rev, sc.Pos, sc.CtxString())
	}
}

func (sc *Scanner) SetStopErrorf(f string, args ...interface{}) {
	s1 := fmt.Sprintf(f, args...)
	s := fmt.Sprintf("%v: %q", s1, sc.CtxString())
	sc.stopErr = &StopError{Pos: sc.Pos, S: s}
}

func (sc *Scanner) SetScanErrorf(f string, args ...interface{}) {
	if sc.scanErrPos > sc.Pos {
		sc.scanErrPos = sc.Pos
		s1 := fmt.Sprintf(f, args...)
		s := fmt.Sprintf("%v: %q", s1, sc.CtxString())
		sc.scanErr = &StopError{Pos: sc.Pos, S: s}
	}
}

//----------

type StopError struct {
	Pos int
	S   string
}

func (se *StopError) Error() string { return se.S }

//----------

type ScanFuncPtr struct {
	fn ScanFunc
}

func (ptr *ScanFuncPtr) Ptr() bool {
	return ptr.fn()
}
func (ptr *ScanFuncPtr) SetPtr(fn ScanFunc) {
	ptr.fn = fn
}

//----------

type ScanFunc func() bool
type RuneFunc func(rune) bool
type ByteFunc func(byte) bool
type Node interface{}
type PosFunc func(int, int) bool

//----------

type ScannerReader interface {
	ReadRuneAt(int) (rune, int, error)
	ReadLastRuneAt(int) (rune, int, error)
}

//----------

//type bytesScannerReader struct {
//	b []byte
//}

//func newBytesScannerReader(b []byte) *bytesScannerReader {
//	return &bytesScannerReader{b: b}
//}
//func (sr *bytesScannerReader) ReadRuneAt(i int) (rune, int, error) {
//	if i > len(sr.b) {
//		return 0, 0, fmt.Errof("bsr: bad index: %v", i)
//	}
//	return DecodeRune(sr.b[i:])
//}
//func (sr *bytesScannerReader) ReadLastRuneAt(i int) (rune, int, error) {
//	if i < 0 {
//		return 0, 0, fmt.Errof("bsr: bad index: %v", i)
//	}
//	return DecodeLastRune(sr.b[:i])
//}
