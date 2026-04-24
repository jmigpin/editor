package btparser

import "fmt"

type ParserState struct {
	src []byte

	UserData   map[string]any
	parseStart Pos
	srcMin     Pos
	srcMax     Pos

	tokDepth int

	// Ignore is used by Token to skip input before matching; set it before parsing and do not modify it while a parse is running.
	Ignore MFn
	ignore struct {
		depth int // recursive call count (avoid loops)
		cache struct {
			valid  bool
			pos    Pos
			result Pos
		}
	}

	farthest Pos
}

func NewParserStateFromBytes(src []byte) *ParserState {
	return &ParserState{src: src, srcMax: Pos(len(src)), UserData: map[string]any{}}
}

func NewParserStateFromString(s string) *ParserState {
	return NewParserStateFromBytes([]byte(s))
}

func (ps *ParserState) Source(mp MPos) []byte {
	start, end := mp.Bounds()
	return ps.src[start:end]
}

func (ps *ParserState) SourceStr(mp MPos) string {
	return string(ps.Source(mp))
}

func (ps *ParserState) Snippet(mp MPos) string {
	return BytesSnippet(ps.src, mp, 30)
}

func (ps *ParserState) sourceLen() Pos {
	return Pos(len(ps.src))
}

//----------
//----------
//----------

type Pos int

// MPos (match position) stores the start position passed to a rule and the end position returned by it; rules may move forward or backward, so End can be lower than Start.
type MPos struct {
	Start Pos
	End   Pos
}

func (mp MPos) Bounds() (Pos, Pos) {
	if mp.Start > mp.End {
		return mp.End, mp.Start
	}
	return mp.Start, mp.End
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
