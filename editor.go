// Source code editor in pure Go.
package main

import (
	"log"
	"os"
	"syscall"

	"github.com/jmigpin/editor/edit"
)

func main() {
	log.SetFlags(log.Llongfile)

	// redirect stderr to log panics
	f, err := os.Create("/home/jorge/editor_stderr.txt")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer f.Close()

	// panic to file
	err = syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
	if err != nil {
		log.Fatalf("Failed to redirect stderr to file: %v", err)
		return
	}

	//mw := io.MultiWriter(f, os.Stdout)
	//log.SetOutput(mw)
	log.SetOutput(os.Stdout)

	_, err = edit.NewEditor()
	if err != nil {
		log.Fatal(err)
	}
}
