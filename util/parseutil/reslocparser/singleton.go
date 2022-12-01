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
var rlps struct {
	once sync.Once
	p    *ResLocParser2
	err  error
}

func getResLocParser() (*ResLocParser2, error) {
	rlps.once.Do(func() {
		rlps.p, rlps.err = newResLocParserSingletonInstance()
	})
	return rlps.p, rlps.err
}
func newResLocParserSingletonInstance() (*ResLocParser2, error) {
	//rlp, err := NewResLocParser()
	//if err != nil {
	//	return nil, err
	//}
	rlp := NewResLocParser2()

	rlp.PathSeparator = rune(os.PathSeparator)
	rlp.Escape = rune(osutil.EscapeRune)
	rlp.ParseVolume = runtime.GOOS == "windows"

	//if err := rlp.Init(false); err != nil {
	//	return nil, err
	//}
	rlp.Init()

	return rlp, nil
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
		Offset:   -1, // TODO
	}
}
