package contentcmd

import (
	"net/url"
	"os/exec"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

// Opens http/https lines in x-www-browser.
func http(ed cmdutil.Editori, row *ui.Row, s string) bool {
	u, err := url.Parse(s)
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
			ed.Error(err)
		}
	}()
	return true
}
