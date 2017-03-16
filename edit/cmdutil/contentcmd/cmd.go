package contentcmd

import (
	"strings"
	"unicode"

	"github.com/jmigpin/editor/edit/cmdutil"
	"github.com/jmigpin/editor/ui"
)

func Cmd(ed cmdutil.Editorer, row *ui.Row) {
	ta := row.TextArea
	// space limited
	s := expandLeftRightUntilSpace(ta.Str(), ta.CursorIndex())
	if ok := openSession(ed, row, s); ok {
		return
	}
	if ok := directory(ed, row, s); ok {
		return
	}
	if ok := fileLine(ed, row, s); ok {
		return
	}
	if ok := http(ed, row, s); ok {
		return
	}
	if ok := goPathDir(ed, row, s); ok {
		return
	}
	// space or quote limited
	s2 := expandLeftRightUntilSpaceOrQuote(ta.Str(), ta.CursorIndex())
	if ok := goPathDir(ed, row, s2); ok {
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
