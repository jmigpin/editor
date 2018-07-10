package core

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug"
	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/core/toolbarparser"
	"github.com/pkg/errors"
)

func GoDebugCmd(erow *ERow, part *toolbarparser.Part) error {
	return fmt.Errorf("todo")

	args := part.ArgsUnquoted()
	return godebugi.Start(erow, args)
}

//----------

var godebugi GoDebugInstance

//----------

type GoDebugInstance struct {
	ed        *Editor
	DataIndex *GDDataIndex
}

func (gdi *GoDebugInstance) Start(erow *ERow, args []string) error {
	if gdi.ed == nil {
		gdi.ed = erow.Ed
	}

	switch args[1] {
	case "run":
		return gdi.run(erow, args)
	case "test":
		return gdi.test(args)
	default:
		return fmt.Errorf("expecting {run,test,find,stop}")
	}
}

//----------

func (gdi *GoDebugInstance) run(erow *ERow, args []string) error {
	if !erow.Info.IsDir() {
		return fmt.Errorf("not a directory")
	}

	erow.Row.TextArea.SetStrClearHistory("")

	erow.Exec.Run(func(ctx context.Context, w io.Writer) error {
		gdi.DataIndex = nil
		gdi.updateUI()

		return gdi.run2(erow, args, ctx, w)
	})
	return nil
}

func (gdi *GoDebugInstance) run2(erow *ERow, args []string, ctx context.Context, w io.Writer) error {
	cmd := godebug.NewCmd(args[1:], nil)
	defer cmd.Cleanup()

	cmd.Dir = erow.Info.Name()
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(ctx); err != nil {
		return err
	}

	// output cmd pid
	fmt.Fprintf(w, "# pid %d\n", cmd.ServerCmd.Process.Pid)

	// handle client msgs loop
	var wg sync.WaitGroup
	wg.Add(1)
	go gdi.clientMsgsLoop(ctx, w, cmd, &wg)
	wg.Wait()

	return cmd.Wait()
}

//----------

func (gdi *GoDebugInstance) clientMsgsLoop(ctx context.Context, w io.Writer, cmd *godebug.Cmd, wg *sync.WaitGroup) {
	defer wg.Done()

	// request file positions before entering loop
	if err := cmd.RequestFileSetPositions(); err != nil {
		fmt.Fprint(w, err)
		//cmd.Stop() // TODO implement cmd.stop?
		return
	}

	// loop
	var updatec <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-cmd.Client.Messages:
			//fmt.Fprintf(w, "client msg %#v\n", msg)
			if !ok {
				// last msg (end of program), final ui update
				gdi.updateUI()
				return
			}
			gdi.handleMsg(msg, w, cmd)
			if updatec == nil {
				t := time.NewTimer(50 * time.Millisecond)
				updatec = t.C
			}

		case <-updatec:
			updatec = nil
			gdi.updateUI()
		}
	}
}

//----------

func (gdi *GoDebugInstance) handleMsg(msg interface{}, w io.Writer, cmd *godebug.Cmd) {
	fmt.Fprintf(w, "handle msg %v\n", msg)

	if gdi.DataIndex == nil {
		gdi.DataIndex = NewGDDataIndex()
	}

	if err := gdi.DataIndex.IndexMsg(msg); err != nil {
		fmt.Fprint(w, err)
		return
	}

	// on receiving the filesdatamsg,  send a requeststart
	if _, ok := msg.(*debug.FilesDataMsg); ok {
		if err := cmd.RequestStart(); err != nil {
			err2 := errors.Wrap(err, "request start")
			fmt.Fprint(w, err2)
			return
		}
	}
}

//----------

func (gdi *GoDebugInstance) updateUI() {
	// keep dataindex at msgs goroutine (safe)
	di := gdi.DataIndex
	if di == nil {
		return
	}

	// run inside ui goroutine with the di pointer
	gdi.ed.UI.RunOnUIGoRoutine(func() {
		for name, i := range di.FilesIndex {
			info, ok := gdi.ed.ERowInfos[name]
			if !ok {
				continue
			}
			info.UpdateAnnotationsRowState(true)
			_ = i
			_ = info
		}
	})
}

//----------

//----------

func (gdi *GoDebugInstance) test(args []string) error {
	return fmt.Errorf("todo")
}

//----------

// GoDebug data Index
type GDDataIndex struct {
	FilesIndex         map[string]int
	Afds               []*debug.AnnotatorFileData
	FileMsgs           []GDFileMsgs
	GlobalArrivalIndex int
}

func NewGDDataIndex() *GDDataIndex {
	di := &GDDataIndex{}
	di.FilesIndex = map[string]int{}
	return di
}

func (di *GDDataIndex) IndexMsg(msg interface{}) error {
	switch t := msg.(type) {
	case *debug.FilesDataMsg:
		di.Afds = t.Data
		// index filenames
		di.FilesIndex = map[string]int{}
		for _, afd := range di.Afds {
			di.FilesIndex[afd.Filename] = afd.FileIndex
		}
		// init index
		di.FileMsgs = make([]GDFileMsgs, len(di.Afds))
		for _, afd := range di.Afds {
			// check index
			if afd.FileIndex > len(di.FileMsgs) {
				return fmt.Errorf("bad file index at init: %v len=%v", afd.FileIndex, len(di.FileMsgs))
			}
			// init
			di.FileMsgs[afd.FileIndex].LineMsgs = make([]GDLineMsgs, afd.DebugLen)
		}
	case *debug.LineMsg:
		// check index
		l1 := len(di.FileMsgs)
		if t.FileIndex >= l1 {
			return fmt.Errorf("bad file index: %v len=%v", t.FileIndex, l1)
		}
		// check index
		l2 := len(di.FileMsgs[t.FileIndex].LineMsgs)
		if t.DebugIndex >= l2 {
			return fmt.Errorf("bad debug index: %v len=%v", t.DebugIndex, l2)
		}
		// index msg
		w := &di.FileMsgs[t.FileIndex].LineMsgs[t.DebugIndex].Msgs
		*w = append(*w, &GDLineMsg{di.GlobalArrivalIndex, t})
		di.GlobalArrivalIndex++
		// mark as having new data
		di.FileMsgs[t.FileIndex].Updated = true
	default:
		return fmt.Errorf("unexpected msg: %T", msg)
	}
	return nil
}

//----------

type GDFileMsgs struct {
	Updated  bool
	LineMsgs []GDLineMsgs
}
type GDLineMsgs struct {
	Msgs []*GDLineMsg
}
type GDLineMsg struct {
	GlobalArrivalIndex int
	LineMsg            *debug.LineMsg
}
