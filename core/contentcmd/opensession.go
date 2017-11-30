package contentcmd

import "github.com/jmigpin/editor/core/cmdutil"

func openSession(erow cmdutil.ERower) bool {
	ta := erow.Row().TextArea
	str := expandLeftRightStopRunes(ta.Str(), ta.CursorIndex(), "")
	if str != "OpenSession" {
		return false
	}
	s2 := afterSpaceExpandRightUntilSpace(ta.Str(), ta.CursorIndex())
	cmdutil.OpenSessionFromString(erow.Ed(), s2)
	return true
}
