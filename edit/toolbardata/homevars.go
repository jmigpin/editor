package toolbardata

import (
	"log"
	"os"
	"strings"
)

var hvars = []string{
//"~", os.Getenv("HOME"),
//"~0", "/home/jorge/projects/golangcode/src/github.com/jmigpin/editor",
}

func init() {
	AppendHomeVar("~", os.Getenv("HOME"))
	AppendHomeVar("~0", "/home/jorge/projects/golangcode/src/github.com/jmigpin/editor")

	//// insert home vars in themselves
	//for i := 0; i < len(hvars); i += 2 {
	//u := InsertHomeVars(hvars[i+1])
	//if u != hvars[i] { // don't replace itself
	//hvars[i+1] = u
	//}
	//}
	log.Println("hvars", hvars)
}

func AppendHomeVar(k, v string) {
	v = removeTrailingSlash(v)
	v = InsertHomeVars(v)
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
	if len(s) >= 2 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
