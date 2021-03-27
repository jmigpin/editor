package goutil

import (
	"context"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
)

//godebug:annotatefile

//----------

func FullEnv() []string {
	env := os.Environ()
	if a, err := GoEnv(); err == nil {
		env = osutil.SetEnvs(env, a)
	}
	return env
}

func GoEnv() ([]string, error) {
	// not the same as os.Environ which has entries like PATH

	args := []string{"go", "env"}
	cmd := osutil.NewCmd(context.Background(), args...)
	bout, err := osutil.RunCmdStdoutAndStderrInErr(cmd, nil)
	if err != nil {
		return nil, err
	}
	env := strings.Split(string(bout), "\n")

	// clear "set " prefix
	if runtime.GOOS == "windows" {
		for i, s := range env {
			env[i] = strings.TrimPrefix(s, "set ")
		}
	}

	env = osutil.UnquoteEnvValues(env)

	return env, nil
}

//----------

func GoRoot() string {
	// doesn't work well in windows
	//return runtime.GOROOT()

	return GetGoRoot(FullEnv())
}

func GoPath() []string {
	return GetGoPath(FullEnv())
}

func GoVersion() (string, error) {
	return GetGoVersion(FullEnv())
}

//----------

func GetGoRoot(env []string) string {
	return osutil.GetEnv(env, "GOROOT")
}

func GetGoPath(env []string) []string {
	//res := []string{}
	//a := osutil.GetEnv(env, "GOPATH")
	//if a != "" {
	//	res = append(res, filepath.SplitList(a)...)
	//} else {
	//	// from go/build/build.go:274
	//	res = append(res, filepath.Join(osutil.HomeEnvVar(), "go"))
	//}
	//return res

	a := osutil.GetEnv(env, "GOPATH")
	return filepath.SplitList(a)
}

// returns version as in "1.0" without the "go" prefix
func GetGoVersion(env []string) (string, error) {
	// get from env var, not present in <=go.15.x?
	v := osutil.GetEnv(env, "GOVERSION")

	if v == "" {
		// get from file located in go root
		d := GetGoRoot(env)
		fp := filepath.Join(d, "VERSION")
		b, err := ioutil.ReadFile(fp)
		if err != nil {
			return "", err
		}
		v = strings.TrimSpace(string(b))
	}

	// remove "go" prefix if present
	v = strings.TrimPrefix(v, "go")

	return v, nil
}

//----------

func AstFileFilename(astFile *ast.File, fset *token.FileSet) (string, error) {
	if astFile == nil {
		panic("!")
	}
	tfile := fset.File(astFile.Package)
	if tfile == nil {
		return "", fmt.Errorf("not found")
	}
	return tfile.Name(), nil
}

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

func Printfc(skip int, f string, args ...interface{}) {
	pc, _, _, ok := runtime.Caller(1 + skip)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		u := details.Name()
		i := strings.Index(u, "(")
		if i > 0 {
			u = u[i:]
		}
		fmt.Printf(u+": "+f, args...)
		return
	}
	fmt.Printf(f, args...)
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
