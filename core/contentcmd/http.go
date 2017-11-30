package contentcmd

import (
	"net/url"
	"os/exec"

	"github.com/jmigpin/editor/core/cmdutil"
)

// Opens http/https lines in x-www-browser.
func http(erow cmdutil.ERower) bool {
	ta := erow.Row().TextArea
	str := expandLeftRightStopRunes(ta.Str(), ta.CursorIndex(), "")
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if !(u.Scheme == "http" || u.Scheme == "https") {
		return false
	}
	go func() {
		cmd := exec.Command("x-www-browser", u.String())
		err := cmd.Run()
		if err != nil {
			ed := erow.Ed()
			ed.Error(err)
			ed.UI().RequestPaint()
		}
	}()
	return true
}
