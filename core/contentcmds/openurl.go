package contentcmds

import (
	"net/url"
	"os/exec"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
)

// Opens url lines in preferred application.
func OpenURL(erow *core.ERow, index int) (bool, error) {
	ta := erow.Row.TextArea

	isHttpRune := func(ru rune) bool {
		extra := parseutil.RunesExcept(parseutil.ExtraRunes, " []()<>")
		return unicode.IsLetter(ru) || unicode.IsDigit(ru) ||
			strings.ContainsRune(extra, ru)
	}

	rd := iorw.NewLimitedReader(ta.TextCursor.RW(), index, index, 1000)
	l, r := parseutil.ExpandIndexesEscape(rd, index, false, isHttpRune, osutil.EscapeRune)

	b, err := rd.ReadNSliceAt(l, r-l)
	if err != nil {
		return false, nil
	}
	str := string(b)

	u, err := url.Parse(str)
	if err != nil {
		return false, nil
	}
	if u.Scheme == "" {
		return false, nil
	}

	ustr := u.String()
	args := []string{"xdg-open", ustr}
	erow.Ed.Messagef("openurl:\n\t%v", strings.Join(args, " "))

	c := exec.Command(args[0], args[1:]...)
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
