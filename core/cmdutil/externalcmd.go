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
				row.Square.SetValue(ui.SquareExecuting, false)
			})
		})
	}()
}
func execRowCmd2(erow ERower, cmd *exec.Cmd) error {

	var wg sync.WaitGroup
	defer wg.Wait()
	pipeToTextarea := func(w *io.Writer) *io.PipeWriter {
		pr, pw := io.Pipe()
		*w = pw
		wg.Add(1)
		go func() {
			defer wg.Done()
			readToERow(pr, erow)
		}()
		return pw
	}

	opw := pipeToTextarea(&cmd.Stdout)
	epw := pipeToTextarea(&cmd.Stderr)
	defer opw.Close()
	defer epw.Close()

	// run command
	err := cmd.Start()
	if err != nil {
		return err
	}
	erow.TextAreaAppendAsync(fmt.Sprintf("# pid %d\n", cmd.Process.Pid))
	return cmd.Wait()
}

// Reads and sends to erow only n frames per second.
// Prevents the editor from hanging with small textarea append requests when the external command is outputting in a tight loop.
// Exits when the reader returns an error (like in close).
func readToERow(reader io.Reader, erow ERower) {
	ch := make(chan string)

	go func() {
		var buf [4 * 1024]byte
		for {
			n, err := reader.Read(buf[:])
			if n > 0 {
				ch <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}
		close(ch)
	}()

	var q []string
	var ticker *time.Ticker
	var timeToSend <-chan time.Time
	for {
		select {
		case s, ok := <-ch:
			if !ok {
				erow.TextAreaAppendAsync(strings.Join(q, ""))
				goto forEnd
			}
			if ticker == nil {
				ticker = time.NewTicker(time.Second / 30)
				timeToSend = ticker.C
				// send first now instead of appending for quick first output
				erow.TextAreaAppendAsync(s)
			} else {
				q = append(q, s)
			}
		case <-timeToSend:
			u := strings.Join(q, "")
			if u != "" {
				erow.TextAreaAppendAsync(u)
			}
			q = []string{}
		}
	}
forEnd:
	if ticker != nil {
		ticker.Stop()
	}
}
