package tautil

import "strings"

func Comment(ta Texta) {
	_ = alterSelectedText(ta, commentLines)
}
func commentLines(str string) (string, bool) {
	altered := false
	for i := 0; i < len(str); {
		// first non space rune
		i0 := strings.IndexFunc(str[i:], isNotSpace)
		if i0 < 0 {
			break // don't touch empty spaces
		} else {
			i += i0
		}
		altered = true
		str = str[:i] + "//" + str[i:] // insert
		i = lineEndIndexNextIndex(str, i)
	}
	if !altered {
		return "", false
	}
	return str, true
}
func Uncomment(ta Texta) {
	_ = alterSelectedText(ta, uncommentLines)
}
func uncommentLines(str string) (string, bool) {
	altered := false
	for i := 0; i < len(str); {
		// first non space rune
		i0 := strings.IndexFunc(str[i:], isNotSpace)
		if i0 < 0 {
			break // don't touch empty spaces
		} else {
			i += i0
		}
		if strings.HasPrefix(str[i:], "//") {
			// remove
			altered = true
			str = str[:i] + str[i+len("//"):]
		}
		i = lineEndIndexNextIndex(str, i)
	}
	if !altered {
		return "", false
	}
	return str, true
}
