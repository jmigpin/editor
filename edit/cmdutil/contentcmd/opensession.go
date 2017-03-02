package contentcmd

import (
	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

func openSession(ed cmdutil.Editori, row *ui.Row, s string) bool {
	if s != "OpenSession" {
		return false
	}
	ta := row.TextArea
	s2 := afterSpaceExpandRightUntilSpace(ta.Str(), ta.CursorIndex())
	cmdutil.OpenSessionFromString(ed, s2)
	return true
}
