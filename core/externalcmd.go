package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jmigpin/editor/core/parseutil"
	"github.com/jmigpin/editor/core/toolbarparser"
)

func ExternalCmd(erow *ERow, part *toolbarparser.Part) {
	cargs := cmdPartArgs(part)
	ExternalCmdFromArgs(erow, cargs, nil)
}

func ExternalCmdFromArgs(erow *ERow, cargs []string, fend func(error)) {
	if erow.Info.IsFileButNotDir() {
		externalCmdFileButNotDir(erow, cargs, fend)
	} else if erow.Info.IsDir() {
		externalCmdDir(erow, cargs, fend, nil)
	} else {
		erow.Ed.Errorf("unable to run external cmd for erow: %v", erow.Info.Name())
	}
}

//----------

// create a row with the file dir and run the cmd
func externalCmdFileButNotDir(erow *ERow, cargs []string, fend func(error)) {
	dir := filepath.Dir(erow.Info.Name())

	info := erow.Ed.ReadERowInfo(dir)
	rowPos := erow.Row.PosBelow()
	erow2 := NewERow(erow.Ed, info, rowPos)

	// position offset: <line:#offset>
	offset := erow.Row.TextArea.TextCursor.Index()
	posOffset := fmt.Sprintf("%v:#%v", erow.Info.Name(), offset)

	// environment
	env := append(
		os.Environ(),
		"edPosOffset="+posOffset,
	)

	externalCmdDir(erow2, cargs, fend, env)
}

//----------

func externalCmdDir(erow *ERow, cargs []string, fend func(error), env []string) {
	if !erow.Info.IsDir() {
		panic("not a directory")
	}

	// cleanup row content
	erow.Row.TextArea.SetStrClearHistory("")
	erow.Row.TextArea.ClearPos()

	fexec := func(ctx context.Context, w io.Writer) error {
		// prepare cmd exec
		cmd := exec.CommandContext(ctx, cargs[0], cargs[1:]...)
		cmd.Dir = erow.Info.Name()
		cmd.Env = env
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		cmd.Stdout = w
		cmd.Stderr = w

		// run command
		err := cmd.Start()
		if err != nil {
			return err
		}

		// ensure kill to child processes on context cancel
		go func() {
			select {
			case <-ctx.Done():
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		}()

		// output pid
		fmt.Fprintf(w, "# pid %d\n", cmd.Process.Pid)

		return cmd.Wait()
	}

	erow.Exec.Run(func(ctx context.Context, w io.Writer) error {
		err := fexec(ctx, w)
		if fend != nil {
			fend(err)
		}
		return err
	})
}

//----------

func cmdPartArgs(part *toolbarparser.Part) []string {
	//if partContainsEscapedPipes(part) {
	//	return shellCmdPartArgs(part)
	//}
	//return directCmdPartArgs(part)
	return shellCmdPartArgs(part)
}

//func partContainsEscapedPipes(part *toolbarparser.Part) bool {
//	for _, a := range part.Args {
//		if a.UnquotedStr() == "\\|" {
//			return true
//		}
//	}
//	return false
//}

//----------

func shellCmdPartArgs(part *toolbarparser.Part) []string {
	str := shellCmdPartArgsStr(part)
	return []string{"sh", "-c", str}
}

func shellCmdPartArgsStr(part *toolbarparser.Part) string {
	var u []string
	for _, a := range part.Args {
		s := a.Str()
		s = parseutil.UnescapeRunes(s, "|")
		u = append(u, s)
	}
	return strings.Join(u, " ")
}

//----------

//func directCmdPartArgs(part *toolbarparser.Part) []string {
//	var u []string
//	for _, a := range part.Args {
//		s := a.UnquotedStr()
//		u = append(u, s)
//	}
//	return u
//}

//----------
