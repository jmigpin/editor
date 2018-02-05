package cmdutil

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/jmigpin/editor/core/toolbardata"
)

func ExternalCmd(erow ERower, part *toolbardata.Part) {
	row := erow.Row()
	ed := erow.Ed()

	// only run commands on directories
	dir := erow.Filename()
	if !erow.IsDir() {
		ed.Errorf("running external cmd on a row that is not a directory: %v", dir)
		return
	}

	// cmd str
	var cmdStr string
	if len(part.Args) == 1 {
		// if quoted it will pass the string inside verbatim
		cmdStr = part.Args[0].UnquotedStr()
	} else {
		// concat args
		var u []string
		for _, a := range part.Args {
			u = append(u, a.Str)
		}
		cmdStr = strings.Join(u, " ")
	}

	// cleanup row content
	row.TextArea.SetStrClear("", true, true)

	// start erow exec state, will clear previous runs if any
	ctx := erow.StartExecState()

	// prepare cmd exec
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// ensure kill to child processes on context cancel
	go func() {
		select {
		case <-ctx.Done():
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()

	// exec
	go func() {
		err := execRowCmd2(erow, cmd)
		erow.ClearExecState(ctx, func() {
			// show error if still on the same context
			if err != nil {
				erow.TextAreaAppendAsync(err.Error())
			}
		})
	}()
}
func execRowCmd2(erow ERower, cmd *exec.Cmd) error {
	// setup output
	w := erow.TextAreaWriter()
	defer w.Close()
	cmd.Stdout = w
	cmd.Stderr = w

	// run command
	err := cmd.Start()
	if err != nil {
		return err
	}
	erow.TextAreaAppendAsync(fmt.Sprintf("# pid %d\n", cmd.Process.Pid))
	return cmd.Wait()
}
