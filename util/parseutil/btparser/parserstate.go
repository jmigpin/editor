package btparser

import "fmt"

type ParserState struct {
	src []byte

	tokenC int
	ignore struct {
		c     int // recursive call count (avoid loops)
		cache struct {
			valid  bool
			pos    Pos
			result Pos
		}
	}

	farthest Pos
}

func NewParserStateFromBytes(src []byte) *ParserState {
	return &ParserState{src: src}
}

func NewParserStateFromString(s string) *ParserState {
	return NewParserStateFromBytes([]byte(s))
}

func (ps *ParserState) Source(mp MPos) []byte {
	return ps.src[mp.Start:mp.End]
}

func (ps *ParserState) SourceStr(mp MPos) string {
	return string(ps.Source(mp))
}

func (ps *ParserState) Snippet(mp MPos) string {
	return BytesSnippet(ps.src, mp, 30)
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

type MFn func(*ParserState, Pos) (MPos, error)
type VFn[T any] func(*ParserState, Pos) (T, MPos, error)

type VHandler[T any] func(T) error
type VMaker[T any] func(MPos) (T, error)

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
