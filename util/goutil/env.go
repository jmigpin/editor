package goutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
)

func OsAndGoEnv(dir string) []string {
	env := os.Environ()
	return osutil.SetEnvs(env, GoEnv(dir))
}

func GoEnv(dir string) []string {
	w, err := GoEnv2(dir)
	if err != nil {
		return nil
	}
	return w
}
func GoEnv2(dir string) ([]string, error) {
	// not the same as os.Environ which has entries like PATH

	args := []string{"go", "env"}
	c := osutil.NewCmdI2(nil, args...)
	c.Cmd().Dir = dir
	bout, err := osutil.RunCmdICombineStderrErr(c)
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

	return GetGoRoot(GoEnv(""))
}

func GoPath() []string {
	return GetGoPath(GoEnv(""))
}

func GoVersion() (string, error) {
	return GetGoVersion(GoEnv(""))
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
