package reslocparser

import (
	"os"
	"runtime"
	"sync"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

func ParseResLoc(src []byte, index int) (*ResLoc, error) {
	rlp, err := getResLocParser()
	if err != nil {
		return nil, err
	}
	return rlp.Parse(src, index)
}
func ParseResLoc2(rd iorw.ReaderAt, index int) (*ResLoc, error) {
	src, err := iorw.ReadFastFull(rd)
	if err != nil {
		return nil, err
	}
	min := rd.Min() // keep to restore position
	rl, err := ParseResLoc(src, index-min)
	if err != nil {
		return nil, err
	}
	// restore position
	rl.Pos += min
	rl.End += min
	return rl, nil
}

//----------

// reslocparser singleton
type resLocParser interface {
	Parse(src []byte, index int) (*ResLoc, error)
}

var rlps struct {
	once sync.Once
	p    resLocParser
	err  error
}

func getResLocParser() (resLocParser, error) {
	rlps.once.Do(func() {
		rlps.p, rlps.err = newResLocParserSingletonInstance()
	})
	return rlps.p, rlps.err
}
func newResLocParserSingletonInstance() (resLocParser, error) {
	escape := rune(osutil.EscapeRune)
	pathSep := rune(os.PathSeparator)
	parseVolume := runtime.GOOS == "windows"

	return NewResLocParser2(escape, pathSep, parseVolume), nil

	// Use this block to switch back to the previous pscan-based parser.
	//rlp := NewResLocParser()
	//rlp.PathSeparator = pathSep
	//rlp.Escape = escape
	//rlp.ParseVolume = parseVolume
	//rlp.Init()
	//return rlp, nil
}

//----------
//----------
//----------

// util func to replace parseutil.*
func ParseFilePos(src []byte, index int) (*parseutil.FilePos, error) {
	rl, err := ParseResLoc(src, index)
	if err != nil {
		return nil, err
	}
	return ResLocToFilePos(rl), nil
}

func ResLocToFilePos(rl *ResLoc) *parseutil.FilePos {
	return &parseutil.FilePos{
		Filename: rl.Path, // original string (unescaped)
		Line:     rl.Line,
		Column:   rl.Column,
		Offset:   rl.Offset,
		Len:      0,
	}
}
