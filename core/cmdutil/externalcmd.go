package cmdutil

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

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

	// indicate the row is running an external cmd
	row.SetState(ui.ExecutingRowState, true)
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
		err := execRowCmd2(erow, cmd)

		// another context could be added already to the row
		row := erow.Row()
		gRowCtx.ClearIfNotNewCtx(row, ctx, func() {
			// show error if any
			if err != nil {
				erow.TextAreaAppendAsync(err.Error())
			}
			// indicate the cmd is not running anymore
			erow.Ed().UI().EnqueueRunFunc(func() {
				row.SetState(ui.ExecutingRowState, false)
			})
		})
	}()
}
func execRowCmd2(erow ERower, cmd *exec.Cmd) error {

	var wg sync.WaitGroup
	defer wg.Wait()
	pipeToTextArea := func(w ...*io.Writer) *io.PipeWriter {
		pr, pw := io.Pipe()
		for _, u := range w {
			*u = pw
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			readToERow(pr, erow)
		}()
		return pw
	}

	opw := pipeToTextArea(&cmd.Stdout, &cmd.Stderr)
	defer opw.Close()

	// run command
	err := cmd.Start()
	if err != nil {
		return err
	}
	erow.TextAreaAppendAsync(fmt.Sprintf("# pid %d\n", cmd.Process.Pid))
	return cmd.Wait()
}

// Reads and sends to erow only n times per second.
func readToERow(reader io.Reader, erow ERower) {
	var buf [64 * 1024]byte
	for {
		n, err := reader.Read(buf[:])
		if n > 0 {
			s := string(buf[:n])
			erow.TextAreaAppendAsync(s)
			// prevent tight loop that can leave UI unresponsive
			time.Sleep(time.Second / (ui.DrawFrameRate - 1))
		}
		if err != nil {
			break
		}
	}
}
