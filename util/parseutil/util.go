package parseutil

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/osutil"
	"github.com/jmigpin/editor/util/scanutil"
)

//----------

func AddEscapes(str string, escape rune, escapeRunes string) string {
	w := []rune{}
	for _, ru := range str {
		if strings.ContainsRune(escapeRunes, ru) {
			w = append(w, escape)
		}
		w = append(w, ru)
	}
	return string(w)
}

func RemoveEscapes(str string, escape rune) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc {
			if ru == escape {
				esc = true
				continue
			}
		} else {
			esc = false
		}
		w = append(w, ru)
	}
	return string(w)
}

// removes the escape only if escapable
func RemoveEscapesEscapable(str string, escape rune, escapable string) string {
	w := []rune{}
	esc := false
	for _, ru := range str {
		if !esc {
			if ru == escape {
				esc = true
				continue
			}
		} else {
			esc = false

			// re-add escape if not one of the escapable
			if !strings.ContainsRune(escapable, ru) {
				w = append(w, escape)
			}
		}
		w = append(w, ru)
	}
	return string(w)
}

//----------

func EscapeFilename(str string) string {
	escape := osutil.EscapeRune
	mustBeEscaped := escapedInFilenames + string(escape)
	return AddEscapes(str, escape, mustBeEscaped)
}

func RemoveFilenameEscapes(f string, escape, pathSep rune) string {
	f = RemoveEscapes(f, escape)
	f = CleanMultiplePathSeps(f, pathSep)
	if u, err := url.PathUnescape(f); err == nil {
		f = u
	}
	return f
}

//----------

func CleanMultiplePathSeps(str string, sep rune) string {
	w := []rune{}
	added := false
	for _, ru := range str {
		if ru == sep {
			if !added {
				added = true
				w = append(w, ru)
			}
		} else {
			added = false
			w = append(w, ru)
		}
	}
	return string(w)
}

//----------

func ExpandIndexesEscape(rd iorw.ReaderAt, index int, truth bool, fn func(rune) bool, escape rune) (int, int) {
	// ensure the index is not in the middle of an escape
	index = ImproveExpandIndexEscape(rd, index, escape)

	l := ExpandLastIndexEscape(rd, index, false, fn, escape)
	r := ExpandIndexEscape(rd, index, false, fn, escape)
	return l, r
}

func ExpandIndexEscape(r iorw.ReaderAt, i int, truth bool, fn func(rune) bool, escape rune) int {
	sc := scanutil.NewScanner(r)
	sc.Pos = i
	return expandEscape(sc, truth, fn, escape)
}

func ExpandLastIndexEscape(r iorw.ReaderAt, i int, truth bool, fn func(rune) bool, escape rune) int {
	sc := scanutil.NewScanner(r)
	sc.Pos = i

	// read direction
	tmp := sc.Reverse
	sc.Reverse = true
	defer func() { sc.Reverse = tmp }() // restore

	return expandEscape(sc, truth, fn, escape)
}

func expandEscape(sc *scanutil.Scanner, truth bool, fn func(rune) bool, escape rune) int {
	for {
		if sc.Match.End() {
			break
		}
		if sc.Match.Escape(escape) {
			continue
		}
		u := sc.Pos
		ru := sc.ReadRune()
		if fn(ru) == truth {
			sc.Pos = u
			break
		}
	}
	return sc.Pos
}

//----------

func ImproveExpandIndexEscape(r iorw.ReaderAt, i int, escape rune) int {
	sc := scanutil.NewScanner(r)
	sc.Pos = i

	// read direction
	tmp := sc.Reverse
	sc.Reverse = true
	defer func() { sc.Reverse = tmp }() // restore

	for {
		if sc.Match.End() {
			break
		}
		if sc.Match.Rune(escape) {
			continue
		}
		break
	}
	return sc.Pos
}

//----------

// Line/col args are one-based.
func LineColumnIndex(rd iorw.ReaderAt, line, column int) (int, error) {
	// must have a good line
	if line <= 0 {
		return 0, fmt.Errorf("bad line: %v", line)
	}
	line-- // make line 0 the first line

	// tolerate bad columns
	if column <= 0 {
		column = 1
	}
	column-- // make column 0 the first column

	index := -1
	l, lStart := 0, 0
	ri := 0
	for {
		if l == line {
			index = lStart // keep line start in case it is a bad col

			c := ri - lStart
			if c >= column {
				index = ri // keep line/col
				break
			}
		} else if l > line {
			break
		}

		ru, size, err := iorw.ReadRuneAt(rd, ri)
		if err != nil {
			// be tolerant about the column
			if index >= 0 {
				return index, nil
			}
			return 0, err
		}
		ri += size
		if ru == '\n' {
			l++
			lStart = ri
		}
	}
	if index < 0 {
		return 0, fmt.Errorf("line not found: %v", line)
	}
	return index, nil
}

// Returned line/col values are one-based.
func IndexLineColumn(rd iorw.ReaderAt, index int) (int, int, error) {
	line, lineStart := 0, 0
	ri := 0
	for ri < index {
		ru, size, err := iorw.ReadRuneAt(rd, ri)
		if err != nil {
			return 0, 0, err
		}
		ri += size
		if ru == '\n' {
			line++
			lineStart = ri
		}
	}
	line++                    // first line is 1
	col := ri - lineStart + 1 // first column is 1
	return line, col, nil
}

//----------

func DetectEnvVar(str, name string) bool {
	vstr := "$" + name
	i := strings.Index(str, vstr)
	if i < 0 {
		return false
	}

	e := i + len(vstr)
	if e > len(str) {
		return false
	}

	// validate rune after the name
	ru, _ := utf8.DecodeRuneInString(str[e:])
	if ru != utf8.RuneError {
		if unicode.IsLetter(ru) || unicode.IsDigit(ru) || ru == '_' {
			return false
		}
	}

	return true
}

//----------

func RunesExcept(runes, except string) string {
	drop := func(ru rune) rune {
		if strings.ContainsRune(except, ru) {
			return -1
		}
		return ru
	}
	return strings.Map(drop, runes)
}

//----------

// Useful to compare src code lines.
func TrimLineSpaces(str string) string {
	return TrimLineSpaces2(str, "")
}

func TrimLineSpaces2(str string, pre string) string {
	a := strings.Split(str, "\n")
	u := []string{}
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" {
			u = append(u, s)
		}
	}
	return pre + strings.Join(u, "\n"+pre)
}

//----------

func UrlToAbsFilename(url2 string) (string, error) {
	u, err := url.Parse(string(url2))
	if err != nil {
		return "", err
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("expecting file scheme: %v", u.Scheme)
	}
	filename := u.Path // unescaped

	if runtime.GOOS == "windows" {
		// remove leading slash in windows returned by url.parse: https://github.com/golang/go/issues/6027
		if len(filename) > 0 && filename[0] == '/' {
			filename = filename[1:]
		}

		filename = filepath.FromSlash(filename)
	}

	if !filepath.IsAbs(filename) {
		return "", fmt.Errorf("filename not absolute: %v", filename)
	}
	return filename, nil
}

func AbsFilenameToUrl(filename string) (string, error) {
	if !filepath.IsAbs(filename) {
		return "", fmt.Errorf("filename not absolute: %v", filename)
	}

	if runtime.GOOS == "windows" {
		filename = filepath.ToSlash(filename)
		// add leading slash to match UrlToAbsFilename behaviour
		if len(filename) > 0 && filename[0] != '/' {
			filename = "/" + filename
		}
	}

	u := &url.URL{Scheme: "file", Path: filename}
	return u.String(), nil // path is escaped
}
