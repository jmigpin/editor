package debug

import (
	"fmt"
	"os"
	"sync"
)

var dsrv struct { // debug server
	sync.Mutex
	srv    *Server
	exited bool
}

//----------

// Called by the generated config.
func StartServer() {
	hotStartServer()
}

func hotStartServer() {
	if dsrv.srv == nil {
		dsrv.Lock()
		if dsrv.srv == nil && !dsrv.exited {
			startServer()
		}
		dsrv.Unlock()
	}
}

func startServer() {
	srv, err := NewServer()
	if err != nil {
		fmt.Printf("error: godebug/debug: start server failed: %v\n", err)
		os.Exit(1)
	}
	dsrv.srv = srv
}

//----------

// Auto-inserted at main for a clean exit. Not to be used.
func ExitServer() {
	dsrv.Lock()
	if !dsrv.exited && dsrv.srv != nil {
		dsrv.srv.Close()
	}
	dsrv.exited = true
	dsrv.Unlock()
}

func Exit(code int) {
	ExitServer()
	os.Exit(code)
}

//----------

// Auto-inserted at annotations. Not to be used.
func Line(fileIndex, debugIndex, offset int, item Item) {
	hotStartServer()
	lmsg := &LineMsg{FileIndex: fileIndex, DebugIndex: debugIndex, Offset: offset, Item: item}
	dsrv.srv.Send(lmsg)
}
