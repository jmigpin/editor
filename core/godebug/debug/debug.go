package debug

import (
	"fmt"
	"os"
	"sync"
)

var server *Server
var startServerMu sync.Mutex

// Called by the generated config.
func StartServer() {
	hotStartServer()
}

func hotStartServer() {
	if server == nil {
		startServerMu.Lock()
		if server == nil {
			startServer()
		}
		startServerMu.Unlock()
	}
}

func startServer() {
	srv, err := NewServer()
	if err != nil {
		fmt.Printf("godebug/debug: start server error: %v\n", err)
		os.Exit(1)
	}
	server = srv
}

//----------

// Auto-inserted at main for a clean exit. Not to be used.
func ExitServer() {
	if server != nil {
		server.Close()
	}
}

//----------

// Auto-inserted at annotations. Not to be used.
func Line(fileIndex, debugIndex, offset int, item Item) {
	if notSending {
		return
	}
	hotStartServer()
	lmsg := &LineMsg{FileIndex: fileIndex, DebugIndex: debugIndex, Offset: offset, Item: item}
	server.Send(lmsg)
}

//----------

/*
Stop sending msgs, allows bypassing a program tight loop that otherwise would take too long to complete
Example:
	func f(){
		debug.SetSend(false)
		defer debug.SetSend(true)
		for i:=0; i<10000;i++{
			... // stmts that would become too slow if debug was on
		}
	}
*/

var notSending bool // not using locks, default is "sending"

func SetSend(v bool) {
	notSending = !v
}
