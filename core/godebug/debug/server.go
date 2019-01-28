package debug

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
)

//var logger = log.New(os.Stdout, "debug: ", 0)
var logger = log.New(ioutil.Discard, "debug: ", 0)

// contains all debug data and is populated at init by a generated config on compile
var AnnotatorFilesData []*AnnotatorFileData

//----------

var ServerNetwork string
var ServerAddress string

//----------

type Server struct {
	ln    net.Listener
	cconn net.Conn // only one client at a time

	running sync.RWMutex

	lnWg, cWg, slWg sync.WaitGroup
	slch            chan interface{} // sending loop channel
}

func NewServer() (*Server, error) {
	logger.Print("listen")

	// start listening
	ln, err := net.Listen(ServerNetwork, ServerAddress)
	if err != nil {
		return nil, err
	}

	srv := &Server{ln: ln}
	srv.slch = make(chan interface{}, 10)

	// start locked (not running, no client)
	srv.running.Lock()

	// accept connections
	srv.lnWg.Add(1)
	go func() {
		defer srv.lnWg.Done()
		srv.acceptClientsLoop()
	}()

	// sending loop
	srv.slWg.Add(1)
	go func() {
		defer srv.slWg.Done()
		srv.sendingLoop()
	}()

	return srv, nil
}

func (srv *Server) Close() {
	logger.Println("closing server")

	// close sending loop before stoping client
	close(srv.slch)
	srv.slWg.Wait()

	// close client before stoping listener
	srv.closeClient()
	srv.cWg.Wait()

	// close listener
	_ = srv.ln.Close()
	srv.lnWg.Wait()
}

func (srv *Server) closeClient() {
	if srv.cconn == nil {
		return
	}

	srv.running.Lock() // stop running

	err := srv.cconn.Close()
	if err != nil {
		logger.Print(err)
	}
	srv.cconn = nil
}

//----------

func (srv *Server) acceptClientsLoop() {
	for {
		logger.Println("waiting for client")

		// accept client
		conn, err := srv.ln.Accept()
		if err != nil {
			logger.Print(err)

			// unable to accept (ex: server was closed)
			if operr, ok := err.(*net.OpError); ok {
				if operr.Op == "accept" {
					break
				}
			}

			continue
		}

		logger.Println("got client")

		// if there was a client, close connection
		srv.closeClient()

		// keep client connection
		srv.cconn = conn

		// receive messages from client
		srv.cWg.Add(1)
		go func() {
			defer srv.cWg.Done()
			srv.receiveClientMsgsLoop()
		}()
	}
}

//----------

func (srv *Server) receiveClientMsgsLoop() {
	for {
		msg, err := DecodeMessage(srv.cconn)
		if err != nil {
			logger.Print(err)

			// unable to read (server was probably closed)
			if operr, ok := err.(*net.OpError); ok {
				if operr.Op == "read" {
					break
				}
			}
			// connection ended gracefully by the client
			if err == io.EOF {
				break
			}

			// always print if the error reaches here
			log.Print(err)
			return
		}

		// handle msg
		switch msg.(type) {
		case *ReqFilesDataMsg:
			logger.Print("sending files data")
			msg := &FilesDataMsg{Data: AnnotatorFilesData}
			srv.send3(msg)
		case *ReqStartMsg:
			logger.Print("running unlocked")
			srv.running.Unlock() // start running
		default:
			// always print if there is a new msg type
			log.Printf("todo: unexpected msg type")
		}
	}
}

//----------

func (srv *Server) sendingLoop() {
	for {
		v, ok := <-srv.slch
		if !ok {
			break
		}
		srv.send2(v)
	}
}

func (srv *Server) send2(v interface{}) {

	srv.send3(v)
}

func (srv *Server) send3(v interface{}) {
	encoded, err := EncodeMessage(v)
	if err != nil {
		panic(err)
	}
	if _, err := srv.cconn.Write(encoded); err != nil {
		logger.Print(err)
	}
}

//----------

func (srv *Server) Send(v interface{}) {
	// wait for start/running
	srv.running.RLock()
	defer srv.running.RUnlock()

	// send order is important
	srv.slch <- v
}
