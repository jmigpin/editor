package contentcmds

import (
	"context"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/statemach"
)

func OpenSession(ctx context.Context, erow *core.ERow, index int) (error, bool) {
	ta := erow.Row.TextArea

	// limit reading
	rw := ta.TextCursor.RW()
	rd := iorw.NewLimitedReader(rw, index, index, 1000)

	sname, err := sessionName(rd, index)
	if err != nil {
		return nil, false
	}

	erow.Ed.UI.RunOnUIGoRoutine(func() {
		core.OpenSessionFromString(erow.Ed, sname)
	})

	return nil, true
}

func sessionName(rd iorw.Reader, index int) (string, error) {
	sc := statemach.NewScanner(rd)
	sc.SetStartPos(index)

	// index at: "OpenSe|ssion sessionname"
	sc.Reverse = true
	_ = sc.Match.FnLoop(sessionNameRune)
	sc.Reverse = false

	// index at: "|OpenSession sessionname"
	sname, err := readCmdSessionName(sc)
	if err == nil {
		// found
		return sname, nil
	}

	// index at: "OpenSession |sessionname"
	sc.Reverse = true
	if !sc.Match.Rune(' ') {
		return "", sc.Errorf("space")
	}
	_ = sc.Match.FnLoop(sessionNameRune)
	sc.Reverse = false

	// index at: "|OpenSession sessionname"
	sname, err = readCmdSessionName(sc)
	if err == nil {
		// found
		return sname, nil
	}

	return "", sc.Errorf("not found")
}

func readCmdSessionName(sc *statemach.Scanner) (string, error) {
	var err error
	ok := sc.RewindOnFalse(func() bool {
		cmd := "OpenSession"
		if !sc.Match.Sequence(cmd + " ") {
			err = sc.Errorf("cmd")
			return false
		}
		sc.Advance()
		if !sc.Match.FnLoop(sessionNameRune) {
			err = sc.Errorf("sessionname")
			return false
		}
		return true
	})
	if !ok {
		return "", err
	}
	return sc.Value(), nil
}

func sessionNameRune(ru rune) bool {
	return unicode.IsLetter(ru) ||
		unicode.IsDigit(ru) ||
		strings.ContainsRune("_-.", ru)
}
