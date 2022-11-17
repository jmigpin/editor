package reslocparser

import (
	"os"
	"runtime"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/parseutil"
)

var rlp *ResLocParser

func initResLocParserSingleton() (*ResLocParser, error) {
	rlp, err := NewResLocParser()
	if err != nil {
		return nil, err
	}

	rlp.PathSeparator = rune(os.PathSeparator)
	rlp.Escape = rune(osutil.EscapeRune)
	rlp.ParseVolume = runtime.GOOS == "windows"

	if err := rlp.Init(false); err != nil {
		return nil, err
	}
	return rlp, nil
}

//----------

func ParseResLoc(src []byte, index int) (*ResLoc, error) {
	// init single instance on demand
	if rlp == nil {
		rlp0, err := initResLocParserSingleton()
		if err != nil {
			return nil, err
		}
		rlp = rlp0
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
