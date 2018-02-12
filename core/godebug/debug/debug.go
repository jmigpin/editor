package debug

import (
	"log"
	"os"
	"sync"
)

var server *Server
var serverHotStartMu sync.Mutex

func Exit() {
	// TODO: on panic wrap main with recover ?
	if server != nil {
		server.Close()
	}
}

func startServer() {
	srv, err := NewServer()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	server = srv
}

func hotStartServer() {
	if server == nil {
		serverHotStartMu.Lock()
		if server == nil {
			startServer()
		}
		serverHotStartMu.Unlock()
	}
}

func Line(fileIndex, debugIndex, offset int, item Item) {
	hotStartServer()
	lmsg := &LineMsg{FileIndex: fileIndex, DebugIndex: debugIndex, Offset: offset, Item: item}
	server.Send(lmsg)
}
