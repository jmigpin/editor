package debug

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// NOTE: init() functions declared across multiple files in a package are processed in alphabetical order of the file name
func init() {
	// runs on editorSide/execSide

	registerStructsForProtoConn()

	if eso.onExecSide {
		es.init()
	}
}

//----------

// exec side: runs before init()s, needed because there could be an Exit() call throught some other init() func, and the initwait must initialize before that to block sending until init is done
var es = newES()

// exec side options (set by generated config)
var eso struct {
	onExecSide bool // on exec side

	addr                Addr
	isServer            bool
	noInitMsg           bool
	srcLines            bool                 // warning at init msg
	syncSend            bool                 // don't send in chunks (slow)
	stringifyBytesRunes bool                 // "abc" instead of [97 98 99]
	filesData           []*AnnotatorFileData // all debug data

	// TODO: currently always true
	//acceptOnlyFirstConn bool // avoid possible hanging progs waiting for another connection to continue debugging (most common case)
}

//----------
//----------
//----------

func StartEditorSide(ctx context.Context, isServer bool, addr Addr) (*Proto, error) {
	eds := &ProtoEditorSide{}
	//eds.logOn = true
	//log.Printf("edside -> %v\n", addr)
	p := NewProto(ctx, isServer, addr, eds)
	err := p.Connect()
	return p, err
}

//----------
//----------
//----------

// exec side
type ES struct {
	p     *Proto
	initw *InitWait
}

func newES() *ES {
	es := &ES{}
	es.initw = newInitWait()
	return es
}
func (es *ES) init() {
	defer es.initw.done()

	if !eso.noInitMsg {
		msg := "binary compiled with editor debug data. Use -noinitmsg to omit this msg."
		if !eso.srcLines {
			msg += fmt.Sprintf(" Note that in the case of panic, the src lines will not correspond to the original src code, but to the annotated src (-srclines=false).")
		}
		execSideLogf("%v\n", msg)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "connectTimeout", 5*time.Second)

	fd := &FilesDataMsg{Data: eso.filesData}
	exs := &ProtoExecSide{fdata: fd, NoWriteBuffering: eso.syncSend}
	//exs.logOn = true
	//log.Printf("exec -> %v\n", eso.addr)
	es.p = NewProto(ctx, eso.isServer, eso.addr, exs)
	if err := es.p.Connect(); err != nil {
		execSideError(err)
	}
}
func (es *ES) afterInitOk(fn func()) {
	mustBeExecSide()
	es.initw.wait()
	if !es.p.GotConnectedFastCheck() {
		return
	}
	fn()
}

//----------

func mustBeExecSide() {
	if !eso.onExecSide {
		panic("not on exec side")
	}
}
func execSideError(err error) {
	execSideLogf("error: %v\n", err)
}
func execSideLogf(f string, args ...any) {
	// TODO: should be exec side only
	fmt.Fprintf(os.Stderr, "DEBUG: "+f, args...)
}

//----------

// Auto-inserted at defer main for a clean exit. Don't use.
func Close() {
	es.afterInitOk(func() {
		if err := es.p.WaitClose(); err != nil {
			execSideError(err)
		}
	})
}

// Auto-inserted in annotated files to replace os.Exit calls. Don't use.
// Non-annotated files that call os.Exit will not let the editor receive all debug msgs. The sync msgs option will need to be used.
func Exit(code int) {
	Close()
	if !eso.noInitMsg {
		execSideLogf("exit code: %v\n", code)
	}
	os.Exit(code)
}

// Auto-inserted at annotations. Don't use.
// NOTE: func name is used in annotator, don't rename.
func L(fileIndex, debugIndex, offset int, item Item) {
	lmsg := &LineMsg{
		FileIndex:  AfdFileIndex(fileIndex),
		DebugIndex: AfdDebugLen(debugIndex),
		Offset:     AfdFileSize(offset),
		Item:       item,
	}
	es.afterInitOk(func() {
		if err := es.p.WriteLineMsg(lmsg); err != nil {
			lineErrOnce.Do(func() {
				execSideError(err)
			})
		}
	})
}

var lineErrOnce sync.Once

//----------
//----------
//----------

type InitWait struct {
	wg       *sync.WaitGroup
	waitSlow bool
}

func newInitWait() *InitWait {
	iw := &InitWait{}
	iw.wg = &sync.WaitGroup{}
	iw.wg.Add(1)
	return iw
}
func (iw *InitWait) wait() {
	if !iw.waitSlow {
		iw.waitSlow = true
		iw.wg.Wait()
	}
}
func (iw *InitWait) done() {
	iw.wg.Done()
}
