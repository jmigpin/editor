package edit

import (
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

func stringCmd(ed *Editor, row *ui.Row) {
	ta := row.TextArea

	s := expandLeftRightUntilSpace(ta.Str(), ta.CursorIndex())

	switch s {
	case "OpenSession":
		s2 := afterSpaceExpandRightUntilSpace(ta.Str(), ta.CursorIndex())
		cmdutil.OpenSessionFromString(ed, s2)
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

	s2 := expandLeftRightUntilSpaceOrQuote(ta.Str(), ta.CursorIndex())
	if ok := stringCmdGoPathDir(ed, row, s2); ok {
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
func expandLeftRightUntilSpaceOrQuote(str string, index int) string {
	if index > len(str) {
		index = len(str)
	}

	isStop := func(ru rune) bool {
		return unicode.IsSpace(ru) || ru == '"'
	}

	i0 := strings.LastIndexFunc(str[:index], isStop)
	if i0 < 0 {
		i0 = 0
	} else {
		i0 += 1 // size of stop rune (quote or space)
	}
	i1 := strings.IndexFunc(str[index:], isStop)
	if i1 < 0 {
		i1 = len(str)
	} else {
		i1 += index
	}
	s2 := str[i0:i1]
	return s2
}

func stringCmdDirectory(ed *Editor, row *ui.Row, cmd string) bool {
	p := cmd
	if !path.IsAbs(cmd) {
		tsd := ed.RowToolbarStringData(row)
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
	row, err = ed.FindRowOrCreateInColFromFilepath(p, col)
	if err == nil {
		row.Square.WarpPointer()
	}
	return true
}

// Opens filename at line, like in compiler errors <string:int> format.
func stringCmdFilenameAndNumber(ed *Editor, row *ui.Row, scmd string) bool {
	// filename
	a := strings.Split(scmd, ":")
	filename := a[0]
	if !path.IsAbs(filename) {
		tsd := ed.RowToolbarStringData(row)
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
	cmdutil.OpenFileLineAtCol(ed, filename, num, row.Col)
	return true
}

// Opens http/https lines in x-www-browser.
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

// Get strings enclosed in quotes, like an import line in a go file, and open the file if found in GOROOT/GOPATH directories.
func stringCmdGoPathDir(ed *Editor, row *ui.Row, s string) bool {
	gopath := os.Getenv("GOPATH")
	a := strings.Split(gopath, ":")
	a = append(a, os.Getenv("GOROOT"))
	for _, p := range a {
		p2 := path.Join(p, "src", s)
		_, err := os.Stat(p2)
		if err == nil {
			col := ed.activeColumn()
			row, err = ed.FindRowOrCreateInColFromFilepath(p2, col)
			if err == nil {
				row.Square.WarpPointer()
			}
			return true
		}
	}
	return false
}
