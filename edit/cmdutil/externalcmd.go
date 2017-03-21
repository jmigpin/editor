package cmdutil

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"

	"github.com/jmigpin/editor/ui"
)

func ExternalCmd(erow ERower, cmdStr string) {
	tsd := erow.ToolbarSD()
	row := erow.Row()
	ed := erow.Editorer()

	// don't run external commands on confirmed files
	_, ok := tsd.FirstPartFilename()
	if ok {
		ed.Error(fmt.Errorf("not running external command on existing filename"))
		return
	}

	dir := ""
	d, ok := tsd.FirstPartDirectory()
	if ok {
		dir = d
	}

	// cancel previous context if any
	gRowCtx.Cancel(row)

	// setup context
	ctx0 := context.Background()
	ctx := gRowCtx.Add(row, ctx0)
	// prepare row
	row.Square.SetValue(ui.SquareExecuting, true)
	row.TextArea.SetStrClear("", true, true)

	// cmd
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
		erow.TextAreaAppend(s)
		// running async, need to request paint
		erow.Editorer().UI().RequestTreePaint()
	}

	var wg sync.WaitGroup
	pipeToChan := func(pr io.Reader) {
		wg.Add(1)
		defer wg.Done()
		b := make([]byte, 5*1024)
		for {
			// when the pipe gets closed, this goroutine gets released
			n, err := pr.Read(b)
			if n > 0 {
				taAppend(string(b[:n]))
			}
			if err != nil {
				break
			}
		}
	}
	// setup piping to the chan
	go pipeToChan(opr)
	go pipeToChan(epr)

	// run command
	err := cmd.Start()
	if err != nil {
		taAppend(err.Error())
	} else {
		taAppend(fmt.Sprintf("# pid %d\n", cmd.Process.Pid))
	}
	_ = cmd.Wait() // this error is going already to the stderr pipe

	opw.Close()
	epw.Close()

	// wait for the pipetochan goroutines to finish
	wg.Wait()

	// another context could be added already to the row
	row := erow.Row()
	gRowCtx.ClearIfNotNewCtx(row, ctx, func() {
		// indicate the cmd is not running anymore
		row.Square.SetValue(ui.SquareExecuting, false)
		row.Square.SetValue(ui.SquareDirty, false)
		row.Col.Cols.Layout.UI.RequestTreePaint()
	})
}
