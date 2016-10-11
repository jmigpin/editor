package edit

import (
	"jmigpin/editor/edit/toolbar"
	"jmigpin/editor/ui"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unicode"
)

func textCmd(ed *Editor, row *ui.Row) {
	ta := row.TextArea

	// TODO: act on selections - click and drag with middle button
	//s := ""
	//if ta.SelectionOn() {
	//a := ta.SelectionIndex()
	//b := ta.CursorIndex()
	//if a > b {
	//a, b = b, a
	//}
	//s = ta.Text()[a:b]
	//} else {
	//s = parseTextCmd(ta.Text(), ta.CursorIndex())
	//}

	s := expandLeftRightUntilSpace(ta.Text(), ta.CursorIndex())

	switch s {
	case "OpenSession":
		s2 := afterSpaceExpandRightUntilSpace(ta.Text(), ta.CursorIndex())
		openSessionFromString(ed, s2)
		return
	}

	if ok := textCmdFilenameAndNumber(ed, row, s); ok {
		return
	}
	if ok := textCmdHttp(ed, row, s); ok {
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
	//println("textcmd: expand1:", s3)
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
	//println("textcmd: expand2:", s3)
	return s3
}

// filename:number (mostly compiler errors)
func textCmdFilenameAndNumber(ed *Editor, row *ui.Row, s string) bool {
	filename, ok := textCmdFilename(row, s)
	if !ok {
		return false
	}
	n, ok := textCmdFilenameNumber(row, s)
	if !ok {
		n = 0 // continue with line zero
	}
	openFileLineAtCol(ed, filename, n, row.Col)
	return true
}

func textCmdFilename(row *ui.Row, tcmd string) (string, bool) {
	a := strings.Split(tcmd, ":")
	filename := a[0]
	if !path.IsAbs(filename) {
		tsd := toolbar.NewStringData(row.Toolbar.Text())
		d, ok := tsd.DirectoryTag()
		if ok {
			filename = path.Join(d, filename)
		} else {
			f, ok := tsd.FilenameTag()
			if ok {
				filename = path.Join(path.Dir(f), filename)
			}
		}
	}
	fi, err := os.Stat(filename)
	if err != nil { // os.IsNotExist(err)
		return "", false
	}
	if fi.IsDir() {
		return "", false
	}
	return filename, true
}
func textCmdFilenameNumber(row *ui.Row, tcmd string) (int, bool) {
	a := strings.Split(tcmd, ":")
	if len(a) < 2 {
		return 0, false
	}
	num, err := strconv.ParseUint(a[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return int(num), true
}
func textCmdHttp(ed *Editor, row *ui.Row, s string) bool {
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
