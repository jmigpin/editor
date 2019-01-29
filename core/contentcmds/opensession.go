package contentcmds

import (
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
)

func OpenSession(erow *core.ERow, index int) (bool, error) {
	ta := erow.Row.TextArea

	// cmd/sessionname runes
	nameRune := func(ru rune) bool {
		return unicode.IsLetter(ru) ||
			unicode.IsDigit(ru) ||
			strings.ContainsRune("_-.", ru)
	}

	// match cmd
	cmd := "OpenSession"
	cmdSpace := cmd + " "
	k := parseutil.ExpandLastIndexFunc(ta.Str()[:index], 50, false, nameRune)
	str := ta.Str()[k:]
	if !strings.HasPrefix(str, cmdSpace) {
		return false, nil
	}

	// match session name
	str2 := str[len(cmdSpace):]
	u := parseutil.ExpandIndexFunc(str2, 50, false, nameRune)
	sname := str2[:u]

	core.OpenSessionFromString(erow.Ed, sname)

	return true, nil
}
