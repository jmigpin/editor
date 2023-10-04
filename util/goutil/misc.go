package goutil

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
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
	h := CallerFileLine(1)
	return fmt.Sprintf(h+f, args...)
}

//----------

func CallerFileLine(skip int) string {
	pc, _, _, ok := runtime.Caller(1 + skip)
	if ok {
		details := runtime.FuncForPC(pc)
		if details != nil {
			file, line := details.FileLine(pc)
			return fmt.Sprintf("%v:%v: ", file, line)
		}
	}
	return "?:?: "
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

func TodoErrorStr(s string) error {
	return TodoErrorSkip(1, s)
}
func TodoError(args ...any) error {
	return TodoErrorSkip(1, args...)
}
func TodoErrorType(t any) error {
	return TodoErrorSkip(1, fmt.Sprintf("%T", t))
}
func TodoErrorf(f string, args ...any) error {
	return TodoErrorSkipf(1, f, args...)
}
func TodoErrorSkipf(skip int, f string, args ...any) error {
	return TodoErrorSkip(1+skip, fmt.Sprintf(f, args...))
}
func TodoErrorSkip(skip int, args ...any) error {
	h := CallerFileLine(1+skip) + "TODO: "
	args2 := append([]any{h}, args...)
	return errors.New(fmt.Sprint(args2...))
}

//----------

func Logf(f string, args ...any) {
	LogSkipf(1, f, args...)
}
func Log(args ...any) {
	LogSkip(1, args...)
}

func LogSkipf(skip int, f string, args ...any) {
	LogSkip(1+skip, fmt.Sprintf(f, args...))
}
func LogSkip(skip int, args ...any) {
	h := CallerFileLine(1 + skip)
	args2 := append([]any{h}, args...)
	fmt.Print(args2...)
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

func ParseFuncDecl(name, src string) (*ast.FuncDecl, error) {
	src2 := "package tmp\n" + src
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "a.go", src2, 0)
	if err != nil {
		return nil, err
	}

	fd := (*ast.FuncDecl)(nil)
	ast.Inspect(astFile, func(n ast.Node) bool {
		if n, ok := n.(*ast.FuncDecl); ok {
			fd = n
		}
		done := fd != nil
		return !done
	})
	if fd == nil {
		return nil, fmt.Errorf("missing func decl")
	}
	return fd, nil
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
