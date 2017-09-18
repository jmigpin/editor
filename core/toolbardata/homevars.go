package toolbardata

import (
	"os"
	"strings"
)

var hvars = []string{}

func init() {
	AppendHomeVar("~", os.Getenv("HOME"))
}

func AppendHomeVar(k, v string) {
	//v = removeTrailingSlash(v)
	v = InsertHomeVars(v)
	if v == "" || v == "~" {
		return
	}
	hvars = append(hvars, k, v)
}
func DeleteHomeVar(k string) {
	for i := 0; i < len(hvars); i += 2 {
		k2, _ := hvars[i], hvars[i+1]
		if k2 == k {
			hvars = append(hvars[:i], hvars[i+2:]...)
		}
	}
}

func InsertHomeVars(s string) string {
	for i := 0; i < len(hvars); i += 2 {
		k, v := hvars[i], hvars[i+1]
		if strings.HasPrefix(s, v) {
			s = k + s[len(v):]
		}
	}
	return s
}
func RemoveHomeVars(s string) string {
	for i := len(hvars) - 2; i >= 0; i -= 2 {
		v, k := hvars[i], hvars[i+1]
		if strings.HasPrefix(s, v) {
			s = k + s[len(v):]
		}
	}
	return s
}

func removeTrailingSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
