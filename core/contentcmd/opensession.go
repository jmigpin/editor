package contentcmd

import (
	"unicode"

	"github.com/jmigpin/editor/core/cmdutil"
)

func openSession(erow cmdutil.ERower) bool {
	ta := erow.Row().TextArea

	// match "OpenSession"
	l, r := expandLeftRightStop(ta.Str(), ta.CursorIndex(), unicode.IsSpace)
	str := ta.Str()[l:r]
	if str != "OpenSession" {
		return false
	}

	// consume simple spaces
	notSimpleSpace := func(ru rune) bool {
		return ru != ' '
	}
	r2 := expandRightStop(ta.Str(), r, notSimpleSpace)

	// get session name
	r3 := expandRightStop(ta.Str(), r2, unicode.IsSpace)
	sname := ta.Str()[r2:r3]

	cmdutil.OpenSessionFromString(erow.Ed(), sname)
	return true
}
