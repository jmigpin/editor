package contentcmd

import "github.com/jmigpin/editor/edit/cmdutil"

func openSession(erow cmdutil.ERower, s string) bool {
	if s != "OpenSession" {
		return false
	}
	ta := erow.Row().TextArea
	s2 := afterSpaceExpandRightUntilSpace(ta.Str(), ta.CursorIndex())
	cmdutil.OpenSessionFromString(erow.Editorer(), s2)
	return true
}
