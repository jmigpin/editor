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

func ExternalCmd(ed Editorer, row *ui.Row, cmdStr string) {
	tsd := ed.RowToolbarStringData(row)

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
	ed.RowCtx().Cancel(row)

	// setup context
	ctx0 := context.Background()
	ctx := ed.RowCtx().Add(row, ctx0)
	// prepare row
	row.Square.SetExecuting(true)
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
		execRowCmd2(ed, ctx, row, cmd)
	}()
}
func execRowCmd2(ed Editorer, ctx context.Context, row *ui.Row, cmd *exec.Cmd) {
	// pipes to read the cmd output
	opr, opw := io.Pipe()
	epr, epw := io.Pipe()
	cmd.Stdout = opw
	cmd.Stderr = epw

	var wg sync.WaitGroup
	pipeToChan := func(pr io.Reader) {
		wg.Add(1)
		defer wg.Done()
		b := make([]byte, 5*1024)
		for {
			// when the pipe gets closed, this goroutine gets released
			n, err := pr.Read(b)
			if n > 0 {
				appendToRowTextArea(row, string(b[:n]))
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
		appendToRowTextArea(row, err.Error())
	} else {
		s := fmt.Sprintf("# pid %d\n", cmd.Process.Pid)
		appendToRowTextArea(row, s)
	}
	_ = cmd.Wait() // this error is going already to the stderr pipe

	opw.Close()
	epw.Close()

	// wait for the pipetochan goroutines to finish
	wg.Wait()

	// another context could be added already to the row
	ed.RowCtx().ClearIfNotNewCtx(row, ctx, func() {
		// indicate the cmd is not running anymore
		row.Square.SetExecuting(false)
		row.Col.Cols.Layout.UI.RequestTreePaint()
	})
}
func appendToRowTextArea(row *ui.Row, s string) {
	ta := row.TextArea

	// append and cap max size
	s = ta.Str() + s
	maxSize := 1024 * 1024 * 10
	if len(s) > maxSize {
		d := len(s) - maxSize
		s = s[d:]
	}
	ta.SetStrClear(s, false, true) // clear undo for massive savings

	// running async, need to request paint
	row.Col.Cols.Layout.UI.RequestTreePaint()
}
