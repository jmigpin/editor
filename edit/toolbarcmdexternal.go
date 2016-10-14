package edit

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"

	"github.com/jmigpin/editor/ui"
)

func ToolbarCmdExternalForRow(ed *Editor, row *ui.Row, cmd string) {
	workDir := ""
	tsd := ed.rowToolbarStringData(row)

	// don't run external commands on confirmed files
	_, ok := tsd.FirstPartFilename()
	if ok {
		ed.Error(fmt.Errorf("not running external command on existing filename"))
		return
	}

	dir, ok := tsd.FirstPartDirectory()
	if ok {
		workDir = dir
	}
	execRowCmd(row, cmd, workDir)
}

func execRowCmd(row *ui.Row, cmdStr string, dir string) {
	// cancel previous context if any
	rowCtx.Cancel(row)
	// setup context
	ctx0 := context.Background()
	ctx2 := rowCtx.Add(row, ctx0)
	// prepare row
	row.Square.SetExecuting(true)
	row.TextArea.ClearStr("", false)
	// cmd
	cmd := exec.CommandContext(ctx2, "sh", "-c", cmdStr)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	// ensure kill to child processes
	go func() {
		select {
		case <-ctx2.Done():
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()

	// exec
	go func() {
		execRowCmd2(ctx2, row, cmd)
		// another context could be added already to the row
		rowCtx.ClearIfCtx(row, ctx2)
	}()
}
func execRowCmd2(ctx context.Context, row *ui.Row, cmd *exec.Cmd) {

	// pipes to read the cmd output
	opr, opw := io.Pipe()
	epr, epw := io.Pipe()
	cmd.Stdout = opw
	cmd.Stderr = epw

	// channel that the pipes will write to, that will output to the row
	ch := make(chan string)
	go func() {
		for {
			s, ok := <-ch
			if !ok {
				break
			}
			appendToRowTextArea(row, s)
		}
	}()

	var wg sync.WaitGroup
	pipeToChan := func(pr io.Reader) {
		wg.Add(1)
		defer wg.Done()
		b := make([]byte, 5*1024)
		for {
			// when the pipe gets closed, this goroutine gets released
			n, err := pr.Read(b)
			if n > 0 {
				ch <- string(b[:n])
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
	}
	//} else {
	//s := fmt.Sprintf("#pid %d\n", cmd.Process.Pid)
	//appendToRowTextArea(row, s)
	//}
	_ = cmd.Wait() // this error is going already to the stderr pipe

	opw.Close()
	epw.Close()
	// wait for the pipetochan goroutines to finish
	wg.Wait()
	// safely close the pipetochan receiving chan
	close(ch)

	// indicate the cmd is not running anymore
	row.Square.SetExecuting(false)
	row.UI.RequestTreePaint()
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
	ta.ClearStr(s, true)

	row.UI.RequestTreePaint()
}
