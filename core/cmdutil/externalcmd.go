package cmdutil

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui"
)

func ExternalCmd(erow ERower, part *toolbardata.Part) {
	row := erow.Row()
	ed := erow.Ed()

	// only run commands on directories
	fp := erow.Filename()
	if !erow.IsDir() {
		ed.Errorf("running external cmd on a row that is not a directory: %v", fp)
		return
	}

	dir := fp

	// cancel previous context if any
	gRowCtx.Cancel(row)

	// setup context
	ctx0 := context.Background()
	ctx := gRowCtx.Add(row, ctx0)
	// prepare row
	row.Square.SetValue(ui.SquareExecuting, true)
	row.TextArea.SetStrClear("", true, true)

	// cmd str
	var u []string
	for _, a := range part.Args {
		u = append(u, a.Str)
	}
	cmdStr := strings.Join(u, " ")

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
		execRowCmd2(erow, ctx, cmd)
	}()
}
func execRowCmd2(erow ERower, ctx context.Context, cmd *exec.Cmd) {
	// pipes to read the cmd output
	opr, opw := io.Pipe()
	epr, epw := io.Pipe()
	cmd.Stdout = opw
	cmd.Stderr = epw

	taAppend := func(s string) {
		erow.TextAreaAppendAsync(s)
	}

	var wg sync.WaitGroup

	readPipe := func(pr io.Reader) {
		defer wg.Done()
		var buf [1 * 1024 * 1024]byte
		for {
			n, err := pr.Read(buf[:])
			if n > 0 {
				taAppend(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}
	// setup piping to the chan
	wg.Add(2)
	go readPipe(opr)
	go readPipe(epr)

	// run command
	err := cmd.Start()
	if err != nil {
		taAppend(err.Error())
	} else {
		taAppend(fmt.Sprintf("# pid %d\n", cmd.Process.Pid))
	}
	_ = cmd.Wait() // this error is going already to the stderr pipe

	// release goroutines reading the pipes
	opw.Close()
	epw.Close()

	// wait for the pipetochan goroutines to finish
	wg.Wait()

	// another context could be added already to the row
	row := erow.Row()
	gRowCtx.ClearIfNotNewCtx(row, ctx, func() {
		// indicate the cmd is not running anymore
		row.Square.SetValue(ui.SquareExecuting, false)
		row.Square.SetValue(ui.SquareEdited, false)
		row.Col.Cols.Layout.UI.RequestPaint()
	})
}
