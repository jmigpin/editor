package contentcmds

import (
	"net/url"
	"os/exec"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
)

// Opens http/https lines in preferred application.
func http(erow *core.ERow, index int) (bool, error) {
	ta := erow.Row.TextArea

	isHttpRune := func(ru rune) bool {
		return unicode.IsLetter(ru) ||
			unicode.IsDigit(ru) ||
			strings.ContainsRune("_\\-\\.\\\\/~&:#@", ru)
	}

	max := 250
	str := ta.Str()
	ri := parseutil.ExpandIndexFunc(str[index:], max, false, isHttpRune) + index
	li := parseutil.ExpandLastIndexFunc(str[:index], max, false, isHttpRune)
	str = str[li:ri]

	u, err := url.Parse(str)
	if err != nil {
		return false, nil
	}
	if !(u.Scheme == "http" || u.Scheme == "https") {
		return false, nil
	}

	c := exec.Command("xdg-open", u.String())
	if err := c.Start(); err != nil {
		return true, err
	}
	go func() {
		if err := c.Wait(); err != nil {
			erow.Ed.Error(err)
		}
	}()

	return true, nil
}
