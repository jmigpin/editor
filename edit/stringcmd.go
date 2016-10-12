package edit

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/ui"
)

func stringCmd(ed *Editor, row *ui.Row) {
	ta := row.TextArea

	s := expandLeftRightUntilSpace(ta.Text(), ta.CursorIndex())

	switch s {
	case "OpenSession":
		s2 := afterSpaceExpandRightUntilSpace(ta.Text(), ta.CursorIndex())
		openSessionFromString(ed, s2)
		return
	}

	if ok := stringCmdDirectory(ed, row, s); ok {
		return
	}
	if ok := stringCmdFilenameAndNumber(ed, row, s); ok {
		return
	}
	if ok := stringCmdHttp(ed, row, s); ok {
		return
	}
}

func expandLeftRightUntilSpace(str string, index int) string {
	if index > len(str) {
		index = len(str)
	}
	i0 := strings.LastIndexFunc(str[:index], unicode.IsSpace)
	if i0 < 0 {
		i0 = 0
	}
	i1 := strings.IndexFunc(str[index:], unicode.IsSpace)
	if i1 < 0 {
		i1 = len(str)
	} else {
		i1 += index
	}
	s2 := str[i0:i1]
	s3 := strings.TrimSpace(s2)
	return s3
}
func afterSpaceExpandRightUntilSpace(str string, index int) string {
	if index > len(str) {
		index = len(str)
	}
	// find space
	i0 := strings.IndexFunc(str[index:], unicode.IsSpace)
	if i0 < 0 {
		return ""
	}
	i0 += index
	// pass all spaces
	isNotSpace := func(ru rune) bool { return !unicode.IsSpace(ru) }
	i2 := strings.IndexFunc(str[i0:], isNotSpace)
	if i2 < 0 {
		return ""
	}
	i2 += i0
	// find space
	i3 := strings.IndexFunc(str[i2:], unicode.IsSpace)
	if i3 < 0 {
		i3 = len(str)
	} else {
		i3 += i2
	}
	s2 := str[i2:i3]
	s3 := strings.TrimSpace(s2)
	return s3
}

func stringCmdDirectory(ed *Editor, row *ui.Row, cmd string) bool {
	p := cmd
	if !path.IsAbs(cmd) {
		tsd := ed.rowToolbarStringData(row)
		d, ok := tsd.FirstPartDirectory()
		if ok {
			p = path.Join(d, p)
		} else {
			f, ok := tsd.FirstPartFilename()
			if ok {
				p = path.Join(path.Dir(f), p)
			}
		}
	}
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	if !fi.IsDir() {
		return false
	}
	col := ed.activeColumn()
	row, err = ed.openFilepath(p, col)
	if err == nil {
		row.Square.WarpPointer()
	}
	return true
}

// filename:number (mostly compiler errors)
func stringCmdFilenameAndNumber(ed *Editor, row *ui.Row, scmd string) bool {
	// filename
	a := strings.Split(scmd, ":")
	filename := a[0]
	if !path.IsAbs(filename) {
		tsd := ed.rowToolbarStringData(row)
		d, ok := tsd.FirstPartDirectory()
		if ok {
			filename = path.Join(d, filename)
		} else {
			f, ok := tsd.FirstPartFilename()
			if ok {
				filename = path.Join(path.Dir(f), filename)
			}
		}
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return false
	}
	// line number
	num := 0
	if len(a) >= 2 {
		v, err := strconv.ParseUint(a[1], 10, 64)
		if err == nil {
			num = int(v)
		}
	}
	// open
	openFileLineAtCol(ed, filename, num, row.Col)
	return true
}

func stringCmdHttp(ed *Editor, row *ui.Row, s string) bool {
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
