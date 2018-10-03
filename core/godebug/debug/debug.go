package debug

import (
	"fmt"
	"os"
	"sync"
)

var server *Server
var startServerMu sync.Mutex

// called by the generated config
func Start() {
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

func Exit() {
	if server != nil {
		server.Close()
	}
}

//----------

func Line(fileIndex, debugIndex, offset int, item Item) {
	hotStartServer()
	lmsg := &LineMsg{FileIndex: fileIndex, DebugIndex: debugIndex, Offset: offset, Item: item}
	server.Send(lmsg)
}
