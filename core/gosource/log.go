package gosource

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type logf func(string, ...interface{})

var LogDepth int
var Logf logf = func(string, ...interface{}) {}
var Dump = func(...interface{}) {}
var Debug = false

func LogDebug() {
	Debug = true
	Dump = spew.Dump
	Logf = CallerDepthLogf
}

func CallerDepthLogf(f string, a ...interface{}) {
	fname := ""
	fpcs := make([]uintptr, 1) // num of callers to get
	n := runtime.Callers(2, fpcs)
	if n != 0 {
		fun := runtime.FuncForPC(fpcs[0] - 1) // get info
		if fun != nil {
			s := fun.Name()
			i := strings.LastIndex(s, ".")
			if i >= 0 {
				s = s[i:]
			}
			fname = s + ": "
		}
	}

	u := append([]interface{}{LogDepth * 4, ""}, a...)
	fmt.Printf("%*s"+fname+f+"\n", u...)
}
