package debug

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

// Initialized by a generated config file when on exec side.
var EncoderId string
var onExecSide bool // has generated config

//----------

// NOTE: init() functions declared across multiple files in a package are processed in alphabetical order of the file name

func init() {
	EncoderId = "editor_eid_001" // static

	//if err := registerStructsForProtoConn(EncoderId); err != nil {
	//	execSidePrintError(err)
	//	os.Exit(1)
	//}
	registerStructsForProtoConn2("")

	if onExecSide {
		es.init()
	}
}

//----------
//----------
//----------

// exec side options (set by generated config)
var eso struct {
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

// exec side
// runs before init()s, needed because there could be an Exit() call throught some other init() func, before main() starts
var es = newES()

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
		logf("%v\n", msg)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "connectTimeout", 5*time.Second)

	fd := &FilesDataMsg{Data: eso.filesData}
	exs := &ProtoExecSide{fdata: fd, NoWriteBuffering: eso.syncSend}
	es.p = NewProto(ctx, eso.isServer, eso.addr, exs)
	if err := es.p.Connect(); err != nil {
		logError(err)
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
	if !onExecSide {
		panic("not on exec side")
	}
}
func logError(err error) {
	logf("error: %v\n", err)
}
func logf(f string, args ...any) {
	// TODO: should be exec side only
	fmt.Fprintf(os.Stderr, "DEBUG: "+f, args...)
}

//----------

// Auto-inserted at defer main for a clean exit. Don't use.
func Close() {
	es.afterInitOk(func() {
		if err := es.p.WaitClose(); err != nil {
			logError(err)
		}
	})
}

// Auto-inserted in annotated files to replace os.Exit calls. Don't use.
// Non-annotated files that call os.Exit will not let the editor receive all debug msgs. The sync msgs option will need to be used.
func Exit(code int) {
	Close()
	if !eso.noInitMsg {
		logf("exit code: %v\n", code)
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
				logError(err)
			})
		}
	})
}

var lineErrOnce sync.Once

//----------
//----------
//----------

func StartEditorSide(ctx context.Context, isServer bool, addr Addr) (*Proto, error) {
	eds := &ProtoEditorSide{}
	p := NewProto(ctx, isServer, addr, eds)
	err := p.Connect()
	return p, err
}

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

//----------
//----------
//----------

func genDigitsStr(n int) string {
	const src = "0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = src[rand.Intn(len(src))]
	}
	return string(b)
}

//----------
//----------
//----------

// NOTE: explanation of the issue with encoder ids
// The defined structs can live in two pkgs:
// - godebugconfig/debug.ReqFilesDataMsg
// - github.com/jmigpin/editor/core/godebug/debug.ReqFilesDataMsg
// in self debug, the 2nd editor registers its struct, but if the encoderId is the same, it will clash with the existing debug struct from godebugconfig, which is a different struct by virtue of pkg location, but will register in the same name (gob panic)
// generated encoder ids solves this, but then a built binary only works with that editor, and, for example, a connect cmd waiting for a binary has to be built with that editor instance (same encoder id)
// registering only once doesn't work, there needs to be 2 registrations, one for the config and other for editor, but after that, down the wire, the config needs to match the editor (generated ids solves this)
// with generated encoder ids, a built binary will only be able to run with this editor instance. On the other hand, it can self debug any part of the editor, including the debug pkg, inside an editor running from another editor instance.

//----------

// implemented
// a single pkg for all purposes would solve this issue, but then having an external program be injected with a package named "github.com/jmigpin/editor/core/godebug/debug" could fail to compile (pkg exists; there might be changes in the structs; editor pkg not in cache and needs to fetch; ...)
// in case of self debug, the editor running the debug session needs to be compatible with the structs being sent by the client

//----------
