package goutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmigpin/editor/util/osutil"
	"golang.org/x/mod/modfile"
)

//----------

func ReadGoMod(ctx context.Context, dir string, env []string) (*modfile.File, error) {
	f, _, err := ParseDirGoMod(dir)
	if err != nil {
		return nil, err
	}
	return f, nil
}

//----------

func GoModInit(ctx context.Context, dir, modPath string, env []string) error {
	args := []string{"go", "mod", "init"}
	if modPath != "" {
		args = append(args, modPath)
	}
	_, err := runGoModCmd(ctx, dir, args, env)
	return err
}

func GoModTidy(ctx context.Context, dir string, env []string) error {
	args := []string{"go", "mod", "tidy"}
	_, err := runGoModCmd(ctx, dir, args, env)
	return err
}

func GoModRequire(ctx context.Context, dir, path string, env []string) error {
	args := []string{"go", "mod", "edit", "-require=" + path}
	_, err := runGoModCmd(ctx, dir, args, env)
	return err
}

func GoModExclude(ctx context.Context, dir, path string, env []string) error {
	args := []string{"go", "mod", "edit", "-exclude=" + path}
	_, err := runGoModCmd(ctx, dir, args, env)
	return err
}

func GoModReplace(ctx context.Context, dir, old, new string, env []string) error {
	//// fails when using directories that contain the version in the name. So it would not allow a downloaded module to be used (contains directories with '@' version in the name).
	//args := []string{"go", "mod", "edit", "-replace=" + old + "=" + new}
	//_, err := runGoModCmd(ctx, dir, args, env)
	//return err

	// simple append to the file (works, but can add repeated strings)
	//return goModReplaceUsingAppend(ctx, dir, old, new)

	f, fname, err := ParseDirGoMod(dir)
	if err != nil {
		return err
	}
	if err := f.AddReplace(old, "", new, ""); err != nil {
		return err
	}
	f.Cleanup()
	b, err := f.Format()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fname, b, 0660)
}

//----------

func runGoModCmd(ctx context.Context, dir string, args []string, env []string) ([]byte, error) {
	c := osutil.NewCmdIShell(ctx, args...)
	c.Cmd().Dir = dir
	c.Cmd().Env = env
	bout, err := osutil.RunCmdICombineStderrErr(c)
	if err != nil {
		return nil, fmt.Errorf("runGoMod: %v (args=%v, dir=%v)", err, args, dir)
	}
	return bout, nil
}

//----------

func FindGoMod(dir string) (string, bool) {
	v := os.Getenv("GOMOD")
	if v != "" {
		if strings.EqualFold(v, os.DevNull) {
			return "", false
		}
		return v, true
	}
	return findFileUp(dir, "go.mod")
}

//func FindGoSum(dir string) (string, bool) {
//	return findFileUp(dir, "go.sum")
//}

func FindGoWork(dir string) (string, bool) {
	v := os.Getenv("GOWORK")
	if strings.EqualFold(v, "off") {
		return "", false
	}
	if v != "" {
		return v, true
	}
	return findFileUp(dir, "go.work")
}

func findFileUp(dir, name string) (string, bool) {
	for {
		fp := filepath.Join(dir, name)
		_, err := os.Stat(fp)
		if err == nil {
			return fp, true
		}
		// parent dir
		oldDir := dir
		dir = filepath.Dir(dir)
		isRoot := oldDir == dir
		if isRoot {
			return "", false
		}
	}
}

//----------

func ParseDirGoMod(dir string) (*modfile.File, string, error) {
	name, b, err := readDirGoModFile(dir)
	if err != nil {
		return nil, "", err
	}
	f, err := modfile.Parse(name, b, nil) // ParseLax will not read replaces's
	if err != nil {
		return nil, "", err
	}
	return f, name, nil
}

func readDirGoModFile(dir string) (string, []byte, error) {
	s := filepath.Join(dir, "go.mod")
	b, err := ioutil.ReadFile(s)
	return s, b, err
}

//----------

//func GoModCreateContent(dir string, content string) error {
//	filename := filepath.Join(dir, "go.mod")
//	f, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//	if _, err := fmt.Fprintf(f, content); err != nil {
//		return err
//	}
//	return nil
//}

//----------

//func goModReplaceUsingAppend(ctx context.Context, dir, old, new string) error {
//	filename := filepath.Join(dir, "go.mod")
//	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//	u := "replace " + old + " => " + new
//	if _, err := f.WriteString("\n" + u + "\n"); err != nil {
//		return err
//	}
//	return nil
//}
