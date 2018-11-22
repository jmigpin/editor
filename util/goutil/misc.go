package goutil

import (
	"os"
	"path/filepath"
)

func GoPath() []string {
	// TODO: use go/build defaultgopath if it becomes public

	a := []string{}

	add := func(b ...string) { a = append(a, b...) }

	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		add(filepath.SplitList(gopath)...)
	} else {
		// from go/build/build.go:270:3
		add(filepath.Join(os.Getenv("HOME"), "go"))
	}

	return a
}
