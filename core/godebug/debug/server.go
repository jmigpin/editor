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
	ln      net.Listener
	cconn   *ClientConn
	running sync.RWMutex
	wg      sync.WaitGroup
}

func NewServer() (*Server, error) {
	logger.Print("listen")
	ln, err := net.Listen(ServerNetwork, ServerAddress)
	if err != nil {
		return nil, err
	}

	srv := &Server{ln: ln}

	// start locked (no client)
	srv.running.Lock()

	// accept connections
	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()
		srv.acceptClientsLoop()
	}()

	return srv, nil
}

func (srv *Server) Close() {
	logger.Println("closing server")

	srv.closeClient()

	// close listener
	_ = srv.ln.Close()

	// wait for the listener to be closed, as well as the client receive loop
	srv.wg.Wait()
}

func (srv *Server) closeClient() {
	if srv.cconn == nil {
		return
	}
	err := srv.cconn.conn.Close()
	if err != nil {
		logger.Print(err)
	}
	srv.cconn = nil
	srv.running.Lock()
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
		srv.cconn = NewClientConn(conn)

		// receive messages from client
		srv.wg.Add(1)
		go func(cconn *ClientConn) {
			defer srv.wg.Done()
			srv.receiveClientMsgsLoop(cconn)
		}(srv.cconn)
	}
}

//----------

func (srv *Server) receiveClientMsgsLoop(cconn *ClientConn) {
	for {
		msg, err := DecodeMessage(cconn.conn)
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

			continue
		}

		// handle msg
		switch msg.(type) {
		case *ReqFilesDataMsg:
			logger.Print("sending files data")
			msg := &FilesDataMsg{Data: AnnotatorFilesData}
			encoded, err := EncodeMessage(msg)
			if err != nil {
				logger.Print(err)
				break
			}
			if _, err := cconn.conn.Write(encoded); err != nil {
				logger.Print(err)
			}
		case *ReqStartMsg:
			logger.Print("running unlocked")
			srv.running.Unlock()
		default:
			logger.Printf("todo: unexpected msg type")
			//spew.Dump(t)
		}
	}
}

//----------

func (srv *Server) Send(v interface{}) {
	// NOTE: send order is important, can't naively make this concurrent

	// wait for start
	srv.running.RLock()
	srv.running.RUnlock()

	// encode msg
	encoded, err := EncodeMessage(v)
	if err != nil {
		logger.Print(err)
		panic(err)
		//return
	}

	// send
	if _, err := srv.cconn.conn.Write(encoded); err != nil {
		logger.Print(err)
		srv.closeClient()
	}
}

//----------

type ClientConn struct {
	conn net.Conn
}

func NewClientConn(conn net.Conn) *ClientConn {
	return &ClientConn{conn: conn}
}
