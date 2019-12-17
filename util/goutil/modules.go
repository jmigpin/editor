package goutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmigpin/editor/util/osutil"
)

//----------

// go.mod structures

type GoMod struct {
	Module  Module
	Go      string
	Require []Require
	Exclude []Module
	Replace []Replace
}

type Module struct {
	Path    string
	Version string
}

type Require struct {
	Path     string
	Version  string
	Indirect bool
}

type Replace struct {
	Old Module
	New Module
}

//----------

func ReadGoMod(ctx context.Context, dir string) (*GoMod, error) {
	args := []string{"go", "mod", "edit", "-json"}
	out, err := runGoModCmd(ctx, dir, "read", args)
	if err != nil {
		return nil, err
	}
	goMod := &GoMod{}
	if err := json.Unmarshal(out, goMod); err != nil {
		return nil, err
	}
	return goMod, nil
}

func GoModReplace(ctx context.Context, dir, old, new string) error {
	// TODO: has problems writing "new" with "@version" (dir name) it will add the string with a space instead of "@" and later go.mod has a parse error
	//args := []string{"go", "mod", "edit", "-replace", old + "=" + new}
	//_, err := runGoModCmd(ctx, dir, "replace", args)
	//return err

	// simple append to the file since using "go mod edit" can produce content with parse errors (TODO: check later go versions)
	return goModReplaceWithAppend(ctx, dir, old, new)
}

func goModReplaceWithAppend(ctx context.Context, dir, old, new string) error {
	filename := filepath.Join(dir, "go.mod")
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	u := "replace " + old + " => " + new
	if _, err := f.WriteString("\n" + u + "\n"); err != nil {
		return err
	}
	return nil
}

func GoModRequire(ctx context.Context, dir, path string) error {
	args := []string{"go", "mod", "edit", "-require=" + path}
	_, err := runGoModCmd(ctx, dir, "require", args)
	return err
}

func runGoModCmd(ctx context.Context, dir, errStr string, args []string) ([]byte, error) {
	ecmd := osutil.ExecCmdCtxWithAttr(ctx, args[0], args[1:]...)
	ecmd.Dir = dir
	sout, serr, err := osutil.ExecCmdRunOutputs(ecmd)
	if err != nil {
		return nil, fmt.Errorf("goutil.gomod(%v): err=(%v), stderr=(%v), stdout=(%v)", errStr, err, string(serr), string(sout))
	}
	return sout, nil
}

//----------

func GoModCreateContent(dir string, content string) error {
	filename := filepath.Join(dir, "go.mod")
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, content); err != nil {
		return err
	}
	return nil
}

//----------

func FindGoMod(dir string) (string, bool) {
	for {
		goMod := filepath.Join(dir, "go.mod")
		_, err := os.Stat(goMod)
		if err == nil {
			return goMod, true
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
