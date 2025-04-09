package goutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
)

func OsAndGoEnv(dir string) []string {
	env := os.Environ()
	return osutil.AppendEnv(env, GoEnv(dir))
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

	c := osutil.NewCmdI2([]string{"go", "env"})
	c = osutil.NewShellCmd(c, false)
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

	// remove "go" prefix if present
	trim := func(s string) string {
		return strings.TrimPrefix(s, "go")
	}

	// get from env var, not present in <=go.15.x?
	if v := osutil.GetEnv(env, "GOVERSION"); v != "" {
		return trim(v), nil
	}

	// get from file located in go root
	d := GetGoRoot(env)
	fp := filepath.Join(d, "VERSION")
	if b, err := ioutil.ReadFile(fp); err == nil {
		v := strings.TrimSpace(string(b))
		return trim(v), nil
	}

	// get from go cmd
	if out, err := exec.Command("go", "version").Output(); err == nil {
		s := string(out)
		v := ""
		// e.g., "go version go1.22.1 linux/amd64"
		if _, err := fmt.Sscanf(s, "go version %s", &v); err == nil {
			return trim(v), nil
		}
	}

	return "", fmt.Errorf("unable to get go version")
}
