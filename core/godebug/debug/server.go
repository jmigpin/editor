package debug

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

// contains all debug data and is populated at init by a generated config on compile
var AnnotatorFilesData []*AnnotatorFileData

//----------

//var logger = log.New(os.Stdout, "debug: ", 0)
var logger = log.New(ioutil.Discard, "debug: ", 0)

var ServerNetwork string
var ServerAddress string

const chunkSendRate = 15  // per second
const sendNowNMsgs = 2048 // don't wait for send rate, send now (memory)
const sendQSize = 512     // msgs queueing to be sent

//----------

type Server struct {
	ln     net.Listener
	lnwait sync.WaitGroup
	client struct {
		sync.RWMutex
		conn *CConn
	}
	sendReady sync.RWMutex
}

func NewServer() (*Server, error) {
	// start listening
	logger.Print("listen")
	ln, err := net.Listen(ServerNetwork, ServerAddress)
	if err != nil {
		return nil, err
	}

	srv := &Server{ln: ln}
	srv.sendReady.Lock() // not ready to send (no client yet)

	// accept connections
	srv.lnwait.Add(1)
	go func() {
		defer srv.lnwait.Done()
		srv.acceptClientsLoop()
	}()

	return srv, nil
}

//----------

func (srv *Server) Close() {
	// close listener
	logger.Println("closing server")
	_ = srv.ln.Close()
	srv.lnwait.Wait()

	// close client
	logger.Println("closing client")
	srv.client.Lock()
	if srv.client.conn != nil {
		srv.client.conn.Close()
		srv.client.conn = nil
	}
	srv.client.Unlock()

	logger.Println("server closed")
}

//----------

func (srv *Server) acceptClientsLoop() {
	for {
		// accept client
		logger.Println("waiting for client")
		conn, err := srv.ln.Accept()
		if err != nil {
			logger.Printf("accept error: (%T) %v ", err, err)

			// unable to accept (ex: server was closed)
			if operr, ok := err.(*net.OpError); ok {
				if operr.Op == "accept" {
					logger.Println("end accept client loop")
					return
				}
			}

			continue
		}
		logger.Println("got client")

		// start client
		srv.client.Lock()
		if srv.client.conn != nil {
			srv.client.conn.Close() // close previous connection
		}
		srv.client.conn = NewCCon(srv, conn)
		srv.client.Unlock()
	}
}

//----------

func (srv *Server) Send(v *LineMsg) {
	// locks if client is not ready to send
	srv.sendReady.RLock()
	defer srv.sendReady.RUnlock()

	srv.client.conn.Send(v)
}

//----------

// Client connection.
type CConn struct {
	srv          *Server
	conn         net.Conn
	rwait, swait sync.WaitGroup
	sendch       chan *LineMsg // sending loop channel
	reqStart     struct {
		sync.Mutex
		start   chan struct{}
		started bool
		closed  bool
	}
}

func NewCCon(srv *Server, nconn net.Conn) *CConn {
	conn := &CConn{srv: srv, conn: nconn}
	conn.sendch = make(chan *LineMsg, sendQSize)
	conn.reqStart.start = make(chan struct{})

	// receive messages
	conn.rwait.Add(1)
	go func() {
		defer conn.rwait.Done()
		conn.receiveMsgsLoop()
	}()

	// send msgs
	conn.swait.Add(1)
	go func() {
		defer conn.swait.Done()
		conn.sendMsgsLoop()
	}()

	return conn
}

func (conn *CConn) Close() {
	conn.reqStart.Lock()
	if conn.reqStart.started {
		// not sendready anymore
		conn.srv.sendReady.Lock()
	}
	conn.reqStart.closed = true
	conn.reqStart.Unlock()

	// close send msgs: can't close receive msgs first (closes client)
	close(conn.reqStart.start) // ok even if it didn't start
	close(conn.sendch)
	conn.swait.Wait()

	// close receive msgs
	_ = conn.conn.Close()
	conn.rwait.Wait()
}

//----------

func (conn *CConn) receiveMsgsLoop() {
	for {
		msg, err := DecodeMessage(conn.conn)
		if err != nil {
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
			if err := conn.send2(msg); err != nil {
				log.Println(err)
			}
		case *ReqStartMsg:
			logger.Print("reqstart")
			conn.reqStart.Lock()
			if !conn.reqStart.started && !conn.reqStart.closed {
				conn.reqStart.start <- struct{}{}
				conn.reqStart.started = true
				conn.srv.sendReady.Unlock()
			}
			conn.reqStart.Unlock()
		default:
			// always print if there is a new msg type
			log.Printf("todo: unexpected msg type")
		}
	}
}

//----------

func (conn *CConn) sendMsgsLoop() {
	// wait for reqstart, or the client won't have the index data
	_, ok := <-conn.reqStart.start
	if !ok {
		return
	}

	//// commented: simple iterative send (slow)
	//for {
	//	v, ok := <-conn.sendch
	//	if !ok {
	//		break
	//	}
	//	if err := conn.send2(v); err != nil {
	//		log.Println(err)
	//	}
	//}
	//return

	// send in chunks (better performance)
	scheduled := false
	timeToSend := make(chan bool)
	msgs := []*LineMsg{}
	sendMsgs := func() {
		if len(msgs) > 0 {
			if err := conn.send2(msgs); err != nil {
				log.Println(err)
			}
			msgs = nil
		}
	}
	for {
		select {
		case v, ok := <-conn.sendch:
			if !ok {
				goto loopEnd
			}
			msgs = append(msgs, v)
			if len(msgs) >= sendNowNMsgs {
				sendMsgs()
			} else if !scheduled {
				scheduled = true
				go func() {
					d := time.Second / time.Duration(chunkSendRate)
					time.Sleep(d)
					timeToSend <- true
				}()
			}
		case <-timeToSend:
			scheduled = false
			sendMsgs()
		}
	}
loopEnd:
	// send last messages if any
	sendMsgs()
}

func (conn *CConn) send2(v interface{}) error {
	encoded, err := EncodeMessage(v)
	if err != nil {
		panic(err)
	}
	n, err := conn.conn.Write(encoded)
	if err != nil {
		return err
	}
	if n != len(encoded) {
		logger.Printf("n!=len(encoded): %v %v\n", n, len(encoded))
	}
	return nil
}

//----------

func (conn *CConn) Send(v *LineMsg) {
	conn.sendch <- v
}
