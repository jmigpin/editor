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

func ReadGoMod(ctx context.Context, dir string, env []string) (*GoMod, error) {
	args := []string{"go", "mod", "edit", "-json"}
	out, err := runGoModCmd(ctx, dir, args, env)
	if err != nil {
		return nil, err
	}
	goMod := &GoMod{}
	if err := json.Unmarshal(out, goMod); err != nil {
		return nil, err
	}
	return goMod, nil
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

func GoModRequire(ctx context.Context, dir, path string, env []string) error {
	args := []string{"go", "mod", "edit", "-require=" + path}
	_, err := runGoModCmd(ctx, dir, args, env)
	return err
}

func GoModTidy(ctx context.Context, dir string, env []string) error {
	args := []string{"go", "mod", "tidy"}
	_, err := runGoModCmd(ctx, dir, args, env)
	return err
}

func GoModReplace(ctx context.Context, dir, old, new string, env []string) error {
	//// TODO: fails when using directories that contain the version in the name. So it would not allow a downloaded module to be used (contains directories with '@' version in the name).
	//args := []string{"go", "mod", "edit", "-replace=" + old + "=" + new}
	//_, err := runGoModCmd(ctx, dir, args, env)
	//return err

	// TODO: "go mod edit -replace" has problems writing "new" with "@version" (dir name) it will add the string with a space instead of "@" and later go.mod has a parse error
	// simple append to the file (TODO: check later go versions)
	return goModReplaceUsingAppend(ctx, dir, old, new)
}

//----------

func runGoModCmd(ctx context.Context, dir string, args []string, env []string) ([]byte, error) {
	bout, err := osutil.RunExecCmdCtxWithAttrAndGetOutputs(ctx, dir, nil, args, env)
	if err != nil {
		return nil, fmt.Errorf("runGoMod error: args=%v, dir=%v, err=%v", args, dir, err)
	}
	return bout, nil
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

func goModReplaceUsingAppend(ctx context.Context, dir, old, new string) error {
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
