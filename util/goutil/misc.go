package goutil

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"log"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"
	"strings"
)

//----------

func PrintAstFile(w io.Writer, fset *token.FileSet, astFile *ast.File) error {
	// TODO: without tabwidth set, it won't output the source correctly

	// print with source positions from original file

	// Fail: has struct fields without spaces "field int"->"fieldint"
	//cfg := &printer.Config{Mode: printer.SourcePos | printer.TabIndent}

	// Fail: has stmts split with comments in the middle
	//cfg := &printer.Config{Mode: printer.SourcePos | printer.TabIndent | printer.UseSpaces}

	cfg := &printer.Config{Mode: printer.SourcePos, Tabwidth: 4}

	return cfg.Fprint(w, fset, astFile)
}

//----------

// frame callers

func Printfc(skip int, f string, args ...interface{}) {
	fmt.Print(Sprintfc(skip, f, args...))
}
func Sprintfc(skip int, f string, args ...interface{}) string {
	pc, _, _, ok := runtime.Caller(1 + skip)
	if ok {
		details := runtime.FuncForPC(pc)
		if details != nil {
			//u := details.Name()
			//i := strings.Index(u, "(")
			//if i > 0 {
			//	u = u[i:]
			//}
			//return fmt.Sprintf(u+": "+f, args...)

			f2, l := details.FileLine(pc)
			u := fmt.Sprintf("%v:%v", f2, l)
			return fmt.Sprintf(u+": "+f, args...)
		}
	}
	return fmt.Sprintf(f, args...)
}

//----------

func SprintFnStack(skip int) string {
	buf := &strings.Builder{}
	for i := 0; ; i++ {
		pc, _, _, ok := runtime.Caller(1 + skip + i)
		if !ok {
			break
		}
		details := runtime.FuncForPC(pc)
		if details == nil {
			break
		}

		u := details.Name()
		if i := strings.Index(u, ".("); i >= 0 {
			if j := strings.Index(u[i+2:], ")"); j >= 0 {
				u = u[i+2+j+1:]
			}
		}

		buf.WriteString(u + "\n")
	}
	return buf.String()
}

//----------

func TodoError() error {
	return fmt.Errorf(Sprintfc(1, "TODO"))
}
func TodoErrorStr(s string) error {
	return fmt.Errorf(Sprintfc(1, "TODO: %v", s))
}
func TodoErrorType(t interface{}) error {
	return fmt.Errorf(Sprintfc(1, "TODO: %T", t))
}

//----------

func Hashable(v any) bool {
	k := reflect.TypeOf(v).Kind()
	return true &&
		//k != reflect.UnsafePointer &&
		//k != reflect.Pointer &&
		k != reflect.Slice &&
		k != reflect.Map &&
		k != reflect.Func
}

//----------

func Trace(n int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(n + 1)
	if !ok {
		return "?", 0, "?"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return file, line, "?"
	}

	return file, line, fn.Name()
}

//----------

func FuncName(f interface{}) string {
	p := reflect.ValueOf(f).Pointer()
	details := runtime.FuncForPC(p)
	return runtimeSimpleName(details.Name())
}
func CallerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return "<?>"
	}
	details := runtime.FuncForPC(pc)
	return runtimeSimpleName(details.Name())
}
func runtimeSimpleName(name string) string {
	u := name
	if i := strings.Index(u, ".("); i >= 0 {
		if j := strings.Index(u[i+2:], ")"); j >= 0 {
			u = u[i+2+j+1:]
		}
	}
	return u
}

//----------

func JoinPathLists(w ...string) string {
	return strings.Join(w, string(os.PathListSeparator))
}

//----------

// go test -cpuprofile cpu.prof -memprofile mem.prof
// go tool pprof cpu.prof
// view with a browser:
// go tool pprof -http=:8000 cpu.prof

var profFile *os.File

func StartCPUProfile() error {
	filename := "cpu.prof"
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	profFile = f
	log.Printf("profile cpu: %v\n", filename)
	return pprof.StartCPUProfile(f)
}

func StopCPUProfile() error {
	pprof.StopCPUProfile()
	return profFile.Close()
}
