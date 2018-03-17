package cmdutil

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jmigpin/editor/core/godebug"
	"github.com/jmigpin/editor/core/godebug/debug"
	"github.com/jmigpin/editor/core/toolbardata"
	"github.com/jmigpin/editor/ui/tautil"
	"github.com/jmigpin/editor/util/drawutil/loopers"
	"github.com/pkg/errors"
)

var noActiveRowErr = fmt.Errorf("no active row")

func GoDebug(ed Editorer, aERow ERower, part *toolbardata.Part) {
	// part arguments
	pa := part.Args
	args := []string{}
	for _, a := range pa {
		args = append(args, a.UnquotedStr())
	}
	GoDebugArgs(ed, aERow, args[1:])
}

func GoDebugArgs(ed Editorer, aERow ERower, args []string) {
	if err := DefaultGoDebugCmd.init(ed, aERow, args); err != nil {
		ed.Error(err)
	}
}

//------------

var DefaultGoDebugCmd goDebugCmd

//------------

type goDebugCmd struct {
	di         *godebug.DataIndex
	selCounter int

	session struct {
		erow ERower
		ctx  context.Context
	}
}

func (gcmd *goDebugCmd) DataIndex() *godebug.DataIndex {
	return gcmd.di
}

func (gcmd *goDebugCmd) init(ed Editorer, aERow ERower, args []string) error {
	// first argument
	first := ""
	if len(args) > 0 {
		first = args[0]
	}

	switch first {
	case "run", "test":
		if aERow == nil {
			return noActiveRowErr
		}
		return gcmd.runOrTest(aERow, args)
	case "find":
		return gcmd.find(ed, aERow, args[1:])
	case "clear":
		// stop session erow
		if gcmd.session.erow != nil {
			gcmd.session.erow.StopExecState()
			gcmd.session.erow = nil
		}
		// wait for session to terminate
		if gcmd.session.ctx != nil {
			<-gcmd.session.ctx.Done()
		}
		// reset data and update editor
		gcmd.resetDataIndex(ed)
		return nil
	default:
		return fmt.Errorf("Usage: GoDebug {run,test,clear,find}")
	}
}

func (gcmd *goDebugCmd) runOrTest(erow ERower, args []string) error {
	if !erow.IsDir() {
		return fmt.Errorf("erow not a directory")
	}

	// cleanup row content
	erow.Row().TextArea.SetStrClear("", true, true)

	// keep row that is running the cmd for stopping with "clear" cmd
	gcmd.session.erow = erow

	// exec
	ctx := erow.StartExecState() // will cancel previous existent ctx
	go func() {
		// ensure only one cmd at a time runs
		// wait for previous cmd context to stop
		if gcmd.session.ctx != nil {
			<-gcmd.session.ctx.Done()
		}
		// setup new session ctx
		sctx, cancel := context.WithCancel(context.Background())
		gcmd.session.ctx = sctx
		defer cancel()

		err := gcmd.runOrTest2(ctx, erow, args)
		erow.ClearExecState(ctx, func() {
			// clean up if it is not a new ctx
			gcmd.session.erow = nil
			gcmd.session.ctx = nil

			if err != nil {
				erow.TextAreaAppendAsync(err.Error())
			}
		})
	}()

	return nil
}
func (gcmd *goDebugCmd) runOrTest2(ctx context.Context, erow ERower, args []string) error {
	cmd := godebug.NewCmd(args, nil)
	defer cmd.Cleanup()

	cmd.Dir = erow.Dir()

	// setup output
	w := erow.TextAreaWriter()
	defer w.Close()
	cmd.Stdout = w
	cmd.Stderr = w

	// init data index
	gcmd.resetDataIndex(erow.Ed())

	// start
	if err := cmd.Start(ctx); err != nil {
		return err
	}

	// output cmd pid
	erow.TextAreaAppendAsync(fmt.Sprintf("# pid %d\n", cmd.ServerCmd.Process.Pid))

	// send initial request
	go func() {
		if err := cmd.RequestFileSetPositions(); err != nil {
			erow.TextAreaAppendAsync(err.Error())
			return
		}
	}()

	// receive messages and update editor
	var clientLoop sync.WaitGroup
	clientLoop.Add(1)
	go func() {
		defer clientLoop.Done()
		msgs := cmd.Client.Messages
		var timeToUpdate <-chan time.Time
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					// final update
					gcmd.updateEditor(erow.Ed())
					goto forEnd
				}
				gcmd.handleMsg(cmd, erow, msg)
				if timeToUpdate == nil {
					timeToUpdate = time.NewTimer(time.Second / 3).C
				}
			case <-timeToUpdate:
				timeToUpdate = nil
				gcmd.updateEditor(erow.Ed())
			}
		}
	forEnd:
	}()

	// wait for the debug to end
	clientLoop.Wait()
	return cmd.Wait()
}

func (gcmd *goDebugCmd) resetDataIndex(ed Editorer) {
	gcmd.di = godebug.NewDataIndex()
	gcmd.selCounter = 0
	gcmd.updateEditor(ed)
}

func (gcmd *goDebugCmd) handleMsg(cmd *godebug.Cmd, erow ERower, msg interface{}) {
	if err := gcmd.di.IndexMsg(msg); err != nil {
		erow.TextAreaAppendAsync(err.Error())
		return
	}

	// after receiving the filesdatamsg,  send a requeststart
	switch msg.(type) {
	case *debug.FilesDataMsg:
		if err := cmd.RequestStart(); err != nil {
			err2 := errors.Wrap(err, "request start")
			erow.TextAreaAppendAsync(err2.Error())
			return
		}
	}

	// update current counter if it was at the last position
	if gcmd.selCounter >= gcmd.di.Counter-2 {
		gcmd.selCounter = gcmd.di.Counter - 1
	}
}

func (gcmd *goDebugCmd) updateEditor(ed Editorer) {
	ed.UI().RunOnUIGoRoutine(func() {
		gcmd.naked_updateEditor(ed)
	})
}

func (gcmd *goDebugCmd) naked_updateEditor(ed Editorer) {
	seen := make(map[string]bool)
	for _, erow := range ed.ERowers() {
		// don't repeat filenames (don't do duplicates)
		if seen[erow.Filename()] {
			continue
		}

		seen[erow.Filename()] = true
		// update unique filename
		gcmd.naked_updateERowAnnotations(erow)
	}
}

func (gcmd *goDebugCmd) updateERowAnnotations(erow ERower) {
	erow.Ed().UI().RunOnUIGoRoutine(func() {
		gcmd.naked_updateERowAnnotations(erow)
	})
}

func (gcmd *goDebugCmd) naked_updateERowAnnotations(erow ERower) {
	ta := erow.Row().TextArea

	// filename must be annotated and hash must match to show annotations
	clear := true
	afd := gcmd.di.AnnotatorFileData(erow.Filename())
	if afd != nil {
		clear = !erow.TextAreaStrHashEqual(afd.FileSize, afd.FileHash)
	}

	// clear annotations
	if clear {
		if ta.Drawer.Args.AnnotationsOpt != nil {
			ta.Drawer.Args.AnnotationsOpt = nil
			ta.CalcChildsBounds()
			ta.MarkNeedsPaint()
			erow.UpdateStateAndDuplicates()
		}
		return
	}

	// TODO: try to keep old entries if nothing changed
	versions := gcmd.di.AnnotationsEntries(erow.Filename(), gcmd.selCounter)
	taEntries := gcmd.versionsToTextAreaEntries(versions)

	ta.SetAnnotationsOrderedEntries(taEntries)
	ta.CalcChildsBounds()
	ta.MarkNeedsPaint()
	erow.UpdateStateAndDuplicates()
}

func (gcmd *goDebugCmd) NakedUpdateERowAnnotations(erow ERower) {
	if gcmd.di != nil {
		gcmd.naked_updateERowAnnotations(erow)
	}
}

func (gcmd *goDebugCmd) find(ed Editorer, aERow ERower, args []string) error {
	// data index
	if gcmd.di == nil {
		return fmt.Errorf("no data index available")
	}

	// flags
	flags := flag.FlagSet{}
	flags.Usage = func() {
		var buf bytes.Buffer
		buf.WriteString("Find debug msgs.\n")
		flags.SetOutput(&buf)
		flags.PrintDefaults()
		ed.Messagef(buf.String())
	}
	allFlag := flags.String("all", "", "find in all debug msgs: first, last, prev, next, +-<n>")
	lineFlag := flags.String("line", "", "find debug msgs located at current erow cursor line: first, last, prev, next")
	//strFlag := flags.String("str", "", "sub string to match in a debug msg")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *allFlag != "" {
		return gcmd.findInAll(ed, *allFlag)
	}
	if *lineFlag != "" {
		if aERow == nil {
			return noActiveRowErr
		}
		ta := aERow.Row().TextArea
		return gcmd.findInLine(aERow, *lineFlag, ta.CursorIndex())
	}

	return nil
}

func (gcmd *goDebugCmd) findInAll(ed Editorer, typ string) error {

	addToCounter := func(n int) error {
		oldc := gcmd.selCounter
		gcmd.selCounter += n
		if gcmd.selCounter < 0 {
			gcmd.selCounter = 0
			if oldc == 0 {
				return fmt.Errorf("already at first counter")
			}
		}
		if gcmd.selCounter >= gcmd.di.Counter {
			gcmd.selCounter = gcmd.di.Counter - 1
			if oldc >= gcmd.di.Counter-1 {
				return fmt.Errorf("already at last counter")
			}
		}
		return nil
	}

	switch typ {
	case "first":
		gcmd.selCounter = 0
	case "last":
		gcmd.selCounter = gcmd.di.Counter - 1
	case "next":
		if err := addToCounter(1); err != nil {
			return err
		}
	case "prev":
		if err := addToCounter(-1); err != nil {
			return err
		}
	default:
		// parse
		n, err := strconv.ParseInt(typ, 10, 64)
		if err != nil {
			return err
		}
		if err := addToCounter(int(n)); err != nil {
			return err
		}
	}

	return gcmd.gotoCounter(ed, gcmd.selCounter)
}

func (gcmd *goDebugCmd) findInLine(erow ERower, typ string, offset int) error {
	debug, err := gcmd.offsetLastDebug(erow, offset)
	if err != nil {
		return err
	}

	switch typ {
	case "first":
		c, ok := gcmd.di.NextDebugVersion(-1, debug)
		if !ok {
			return fmt.Errorf("no next msgs were found")
		}
		gcmd.selCounter = c
	case "last":
		c, ok := gcmd.di.PreviousDebugVersion(gcmd.di.Counter, debug)
		if !ok {
			return fmt.Errorf("no previous msgs were found")
		}
		gcmd.selCounter = c
	case "next":
		c, ok := gcmd.di.NextDebugVersion(gcmd.selCounter, debug)
		if !ok {
			return fmt.Errorf("no next msgs were found")
		}
		gcmd.selCounter = c
	case "prev":
		c, ok := gcmd.di.PreviousDebugVersion(gcmd.selCounter, debug)
		if !ok {
			return fmt.Errorf("no previous msgs were found")
		}
		gcmd.selCounter = c
	default:
		panic("todo")
	}

	// TODO: need to update row of the previous counter (could be in another file)
	//gcmd.updateERowAnnotations(erow)
	// TODO: update previous counter erow?
	// update editor to update previous counter that could be in another file
	gcmd.updateEditor(erow.Ed())
	return nil
}

func (gcmd *goDebugCmd) gotoCounter(ed Editorer, counter int) error {
	// offset msg at counter
	lmsg := gcmd.di.LineMsgAtCounter(counter)
	if lmsg == nil {
		return fmt.Errorf("line msg not found at counter: %v", counter)
	}
	// afd from file index
	afd := gcmd.di.AnnotatorFileDataFromFileIndex(lmsg.FileIndex)
	if afd == nil {
		return fmt.Errorf("afd not found for file index: %v", lmsg.FileIndex)
	}

	// find erow that matches afd
	erows := ed.FindERowers(afd.Filename)
	if len(erows) == 0 {
		// TODO: open file?
		return fmt.Errorf("file not open: %v", afd.Filename)
	}
	erow := erows[0]

	index := lmsg.Offset

	// goto index
	ed.UI().RunOnUIGoRoutine(func() {

		row := erow.Row()
		ta := row.TextArea

		row.ResizeTextAreaIfVerySmall()
		ta.SetSelectionOff()
		ta.SetCursorIndex(index)
		ta.MakeIndexVisibleAtCenter(index)
		row.TextArea.FlashIndexLine(index)

		// update all files to reflect the counter change that could affect annotations in all files
		gcmd.updateEditor(ed)
	})

	return nil
}

//------------

func (gcmd *goDebugCmd) PrintAnnotation(erow ERower, index, indexOffset int) {
	version, err := gcmd.itemIndexVersion(erow, gcmd.selCounter, index)
	if err != nil {
		erow.Ed().Error(err)
		return
	}

	str := godebug.StringifyItemOffset(version.LineMsg.Item, indexOffset)
	if str == "" {
		return
	}

	// unquote if possible
	s, err := strconv.Unquote(str)
	if err == nil {
		str = s
	}

	erow.Ed().Messagef("godebug:\n%v", str)
}

func (gcmd *goDebugCmd) PreviousAnnotation(erow ERower, itemIndex int) {
	if err := gcmd.previousAnnotation2(erow, itemIndex); err != nil {
		erow.Ed().Error(err)
	}
}
func (gcmd *goDebugCmd) previousAnnotation2(erow ERower, itemIndex int) error {
	// find in all
	if itemIndex < 0 {
		return gcmd.findInAll(erow.Ed(), "prev")
	}
	// find in line
	version, err := gcmd.itemIndexVersion(erow, gcmd.selCounter, itemIndex)
	if err != nil {
		return err
	}
	return gcmd.findInLine(erow, "prev", version.LineMsg.Offset)
}

func (gcmd *goDebugCmd) NextAnnotation(erow ERower, itemIndex int) {
	if err := gcmd.nextAnnotation2(erow, itemIndex); err != nil {
		erow.Ed().Error(err)
	}
}
func (gcmd *goDebugCmd) nextAnnotation2(erow ERower, itemIndex int) error {
	// find in all
	if itemIndex < 0 {
		return gcmd.findInAll(erow.Ed(), "next")
	}
	// find in line
	version, err := gcmd.itemIndexVersion(erow, gcmd.selCounter, itemIndex)
	if err != nil {
		return err
	}
	return gcmd.findInLine(erow, "next", version.LineMsg.Offset)
}

//------------

func (gcmd *goDebugCmd) itemIndexVersion(erow ERower, counter, itemIndex int) (*godebug.DIVersion, error) {
	versions := gcmd.di.AnnotationsEntries(erow.Filename(), counter)
	if itemIndex >= len(versions) {
		return nil, fmt.Errorf("item index not found: %v (len=%v)", itemIndex, len(versions))
	}
	v := versions[itemIndex]
	if v == nil {
		return nil, fmt.Errorf("item index not set: %v (counter=%v)", itemIndex, counter)
	}
	return v, nil
}

func (gcmd *goDebugCmd) offsetLastDebug(erow ERower, offset int) (godebug.DIDebug, error) {
	ta := erow.Row().TextArea
	str := ta.Str()

	// start/end of the line indexes
	si := tautil.LineStartIndex(str, offset)
	ei, nl := tautil.LineEndIndexNextIndex(str, offset)
	if nl {
		ei--
	}

	// search debug msgs in line indexes
	debugs := gcmd.di.LineMsgsBetweenOffsets(erow.Filename(), si, ei)
	if len(debugs) == 0 {
		return nil, fmt.Errorf("no messages found in line: indexes %v, %v", si, ei)
	}

	// use last debug of the line
	debug := debugs[len(debugs)-1]

	return debug, nil
}

func (gcmd *goDebugCmd) versionsToTextAreaEntries(versions []*godebug.DIVersion) []*loopers.AnnotationsEntry {
	entries := make([]*loopers.AnnotationsEntry, len(versions))
	for i, version := range versions {
		// msg not received yet
		if version == nil {
			continue
		}

		str := godebug.StringifyItem(version.LineMsg.Item)
		ae := &loopers.AnnotationsEntry{version.LineMsg.Offset, str}
		entries[i] = ae
	}
	return entries
}
