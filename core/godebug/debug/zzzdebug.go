package debug

import (
	"context"
	"fmt"
	"io"
	"os"
	rdebug "runtime/debug"
	"sync"
	"time"
)

////godebug:annotatepackage

// NOTE: init() functions declared across multiple files in a package are processed in alphabetical order of the file name
func init() {
	// runs on editorSide/execSide

	registerStructsForProtoConn()

	if exso.onExecSide {
		exs.init()
	}
}

//----------

// exec side options (set by generated config)
var exso struct {
	testing bool // not the same as "godebug test"

	onExecSide bool // on exec side

	addr                Addr
	isServer            bool
	continueServing     bool
	noDebugMsg          bool
	srcLines            bool                 // warning at init msg
	syncSend            bool                 // don't send in chunks (slow)
	stringifyBytesRunes bool                 // "abc" instead of [97 98 99]
	filesData           []*AnnotatorFileData // all debug data
}

//----------
//----------
//----------

// exec side: runs before init()s, needed because there could be an Exit() call throught some other init() func, and the initwait must initialize before that to block sending until init is done
var exs = newExecSide()

type execSide struct {
	p     Proto
	initw *InitWait
	logw  io.Writer
}

func newExecSide() *execSide {
	exs := &execSide{}
	exs.initw = newInitWait()
	return exs
}
func (exs *execSide) init() {
	defer exs.initw.done()
	if !exso.noDebugMsg {
		exs.logw = NewPrefixWriter(os.Stderr, "# godebug.exec: ")
	}
	if err := exs.init2(); err != nil {
		exs.logError(err)
		return
	}
	exs.initw.ok = true
}
func (exs *execSide) init2() error {
	if !exso.noDebugMsg {
		exs.logf("binary compiled with editor debug data. Use -nodebugmsg to omit these msgs.\n")
		if !exso.srcLines {
			exs.logf("Note that in the case of panic, the src lines will not correspond to the original src code, but to the annotated src (-srclines=false).\n")
		}
	}

	// initial connect timeout
	ctx := context.Background()
	timeout := 30 * time.Second
	if exso.testing {
		timeout = 500 * time.Millisecond
	}
	ctx = context.WithValue(ctx, "connectTimeout", timeout)

	fd := &FilesDataMsg{Data: exso.filesData}
	pexs := &ProtoExecSide{FData: fd, NoWriteBuffering: exso.syncSend}
	//pexs.Logger = Logger{"pexs: ", exs.logw} // DEBUG: lots of output

	p, err := NewProto(ctx, exso.addr, pexs, exso.isServer, exso.continueServing, exs.logw)
	exs.p = p
	return err
}
func (exs *execSide) afterInitOk(fn func()) {
	exs.initw.afterInitOk(fn)
}

func (exs *execSide) logf(f string, args ...any) {
	if exs.logw != nil {
		fmt.Fprintf(exs.logw, f, args...)
	}
}
func (exs *execSide) logError(err error) {
	exs.logf("error: %v\n", err)
}

//----------
//----------
//----------

// Auto-inserted at functions to recover from panics. Don't use.
func Recover() {
	//mustBeExecSide() // commented for performance

	if r := recover(); r != nil {
		Close()
		//exs.logf("panic (closed): %v\n", r)
		fmt.Fprintf(os.Stderr, "panic (closed): %v\n", r)
		rdebug.PrintStack()
		os.Exit(1)
	}
}

// Auto-inserted at defer main for a clean exit. Don't use.
func Close() {
	mustBeExecSide()
	exs.afterInitOk(func() {
		if err := exs.p.CloseOrWait(); err != nil {
			exs.logError(err)
		}
	})
}

// Auto-inserted in annotated files to replace os.Exit calls. Don't use.
// Non-annotated files that call os.Exit will not let the editor receive all debug msgs. The sync msgs option will need to be used.
func Exit(code int) {
	mustBeExecSide()
	Close()
	exs.logf("exit code: %v\n", code)
	os.Exit(code)
}

// Auto-inserted at annotations. Don't use.
// NOTE: func name is used in annotator, don't rename.
func L(fileIndex, debugIndex, offset int, item Item) {
	//mustBeExecSide() // commented for performance

	lmsg := &OffsetMsg{
		FileIndex: AfdFileIndex(fileIndex),
		MsgIndex:  AfdMsgIndex(debugIndex),
		Offset:    AfdFileSize(offset),
		Item:      item,
	}
	exs.afterInitOk(func() {
		if err := exs.p.WriteMsg(lmsg); err != nil {
			lineErrOnce.Do(func() {
				exs.logError(err)
			})
		}
	})
}

var lineErrOnce sync.Once

//----------
//----------
//----------

type InitWait struct {
	wg   *sync.WaitGroup
	fast bool
	ok   bool
}

func newInitWait() *InitWait {
	iw := &InitWait{}
	iw.wg = &sync.WaitGroup{}
	iw.wg.Add(1)
	return iw
}
func (iw *InitWait) wait() {
	if !iw.fast {
		iw.wg.Wait()
		iw.fast = true
	}
}
func (iw *InitWait) afterInitOk(fn func()) {
	iw.wait()
	if iw.ok {
		fn()
	}
}
func (iw *InitWait) done() {
	iw.wg.Done()
}

//----------

func mustBeExecSide() {
	if !exso.onExecSide {
		panic("not on exec side")
	}
}
