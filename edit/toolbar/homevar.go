package toolbar

import (
	"os"
	"strings"
)

func ReplaceHomeVar(s string) string {
	home := os.Getenv("HOME")
	if strings.HasPrefix(s, home) {
		return "~" + s[len(home):]
	}
	return s
}
func InsertHomeVar(s string) string {
	if strings.HasPrefix(s, "~") {
		home := os.Getenv("HOME")
		return strings.Replace(s, "~", home, 1)
	}
	return s
}
