package contentcmds

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/jmigpin/editor/core"
	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
)

// Opens url lines in preferred application.
func OpenURL(ctx context.Context, erow *core.ERow, index int) (error, bool) {
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
		return err, false // not handled
	}
	str := string(b)

	u, err := url.Parse(str)
	if err != nil {
		return err, false
	}

	// accepted schemes
	switch u.Scheme {
	case "http", "https", "ftp", "mailto":
		// ok
	default:
		err := fmt.Errorf("unsupported scheme: %v", u.Scheme)
		return err, false
	}

	// cmd timeout
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// cmd
	ustr := u.String()
	args := []string{"xdg-open", ustr}
	cmd := osutil.NewCmd(ctx2, args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Start(); err != nil {
		return err, true
	}

	erow.Ed.Messagef("openurl:\n\t%v", strings.Join(args, " "))

	err = cmd.Wait()
	if err != nil {
		err = fmt.Errorf("%v: %v", err, out.String())
	}
	return err, true
}
