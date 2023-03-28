package debug

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

// init() functions declared across multiple files in a package are processed in alphabetical order of the file name
func init() {
	if hasGenConfig {
		RegisterStructsForEncodeDecode(encoderId)
		StartServer()
	}
}

//----------

// Vars populated at init (generated at compile).
var hasGenConfig bool
var encoderId string
var ServerNetwork string
var ServerAddress string
var annotatorFilesData []*AnnotatorFileData // all debug data

var syncSend bool              // don't send in chunks (usefull to get msgs before crash)
var acceptOnlyFirstClient bool // avoid possible hanging progs waiting for another connection to continue debugging (most common case)
var stringifyBytesRunes bool   // output "abc" instead of [97 98 99]
var hasSrcLines bool           // in case of panic, show warning about srclines

//----------

// var logger = log.New(os.Stdout, "debug: ", log.Llongfile)
// var logger = log.New(os.Stdout, "debug: ", 0)
var logger = log.New(ioutil.Discard, "debug: ", 0)

const chunkSendRate = 15       // per second
const chunkSendQSize = 512     // msgs queueing to be sent
const chunkSendNowNMsgs = 2048 // don't wait for send rate, send now (memory)

//----------

type Server struct {
	listen struct {
		ln       net.Listener
		loopWait sync.WaitGroup
	}
	conn struct { // currently only handling one conn at a time
		sync.Mutex
		haveConn *sync.Cond // cc!=nil
		cc       *CConn
	}
	toSend       chan interface{}
	sendLoopWait sync.WaitGroup
}

func NewServer() (*Server, error) {
	srv := &Server{}
	srv.conn.haveConn = sync.NewCond(&srv.conn)

	// start listening
	ln, err := net.Listen(ServerNetwork, ServerAddress)
	if err != nil {
		return nil, err
	}
	srv.listen.ln = ln
	logger.Print("listening")

	// setup sending
	qsize := chunkSendQSize
	if syncSend {
		qsize = 0
	}
	srv.toSend = make(chan interface{}, qsize)
	srv.sendLoopWait.Add(1)
	go func() {
		defer srv.sendLoopWait.Done()
		srv.sendLoop()
	}()

	// accept connections
	srv.listen.loopWait.Add(1)
	go func() {
		defer srv.listen.loopWait.Done()
		srv.acceptClientLoop()
	}()

	// wait for one client to be sendready before returning
	srv.waitForCConnReady()
	logger.Println("server init done (client ready)")

	return srv, nil
}

//----------

func (srv *Server) acceptClientLoop() {
	defer logger.Println("end accept client loop")
	for {
		// accept client
		logger.Println("waiting for client")
		conn, err := srv.listen.ln.Accept()
		if err != nil {
			logger.Printf("accept error: (%T) %v", err, err)

			// unable to accept (ex: server was closed)
			if operr, ok := err.(*net.OpError); ok {
				if operr.Op == "accept" {
					return
				}
			}

			continue
		}

		logger.Println("got client")
		srv.handleConn(conn)

		// don't receive anymore connections
		if acceptOnlyFirstClient {
			logger.Println("no more clients (accepting only first)")
			return
		}
	}
}

//----------

func (srv *Server) handleConn(conn net.Conn) {
	srv.conn.Lock()
	defer srv.conn.Unlock()

	// currently only handling one client at a time
	if srv.conn.cc != nil { // already connected, reject
		_ = conn.Close()
		return
	}

	srv.conn.cc = NewCConn(srv, conn)
	srv.conn.haveConn.Broadcast()
}
func (srv *Server) closeConnection() error {
	srv.conn.Lock()
	defer srv.conn.Unlock()
	if srv.conn.cc != nil {
		cc := srv.conn.cc
		srv.conn.cc = nil // remove connection from server
		return cc.close()
	}
	return nil
}

//----------

func (srv *Server) waitForCConn() *CConn {
	cc := srv.conn.cc
	if cc == nil {
		srv.conn.Lock()
		defer srv.conn.Unlock()
		for srv.conn.cc == nil {
			srv.conn.haveConn.Wait()
		}
		cc = srv.conn.cc
	}
	return cc
}

//----------

func (srv *Server) Send(v *LineMsg) {
	// simply add to a channel to allow program to continue
	srv.toSend <- v
}
func (srv *Server) sendLoop() {
	for {
		v := <-srv.toSend
		logger.Printf("tosend: %#v", v)
		switch t := v.(type) {
		case bool:
			if t == false {
				logger.Println("stopping send loop")
				return
			}
		case *LineMsg:
			cc := srv.waitForCConn()
			cc.sendWhenReady(t)
		}
	}
}

//----------

func (srv *Server) waitForCConnReady() {
	cc := srv.waitForCConn()
	cc.runWhenReady(func() {})
}

//----------

func (srv *Server) Close() error {
	logger.Println("closing server")
	defer logger.Println("server closed")

	// stop accepting more connections
	err := srv.listen.ln.Close()
	srv.listen.loopWait.Wait()

	// stop getting values to send
	srv.toSend <- false
	srv.sendLoopWait.Wait()

	_ = srv.closeConnection()

	return err
}

//----------
//----------
//----------

// Client connection.
type CConn struct {
	srv  *Server
	conn net.Conn

	state struct {
		sync.Mutex
		ready   *sync.Cond // have exchanged init data and can send
		started bool
		closed  bool
	}

	lines  []*LineMsg // inside lock if sending in chunks (vs syncsend)
	chunks struct {
		sync.Mutex
		scheduled bool
		sendWait  sync.WaitGroup
		sent      time.Time
	}
}

func NewCConn(srv *Server, conn net.Conn) *CConn {
	cc := &CConn{srv: srv, conn: conn}
	cc.state.ready = sync.NewCond(&cc.state)

	// receive messages
	go cc.receiveMsgsLoop()

	return cc
}

func (cc *CConn) close() error {
	logger.Println("closing client")
	defer logger.Println("client closed")

	cc.state.Lock()
	defer cc.state.Unlock()
	if cc.state.closed {
		return nil
	}
	cc.state.closed = true

	// wait for last msgs to be sent
	cc.chunks.sendWait.Wait()

	return cc.conn.Close() // stops receiveMsgsLoop
}

func (cc *CConn) closeFromServerWithErr(err error) {
	err2 := cc.srv.closeConnection()
	if err == nil {
		err = err2
	}

	// connection ended gracefully by the client
	if errors.Is(err, io.EOF) {
		return
	}
	// unable to read (client was probably closed)
	if operr, ok := err.(*net.OpError); ok {
		if operr.Op == "read" {
			return
		}
	}
	// always print if the error reaches here
	panic(err)
	logger.Printf("error: %s", err)
	return
}

//----------

func (cc *CConn) receiveMsgsLoop() {
	for {
		msg, err := DecodeMessage(cc.conn)
		if err != nil {
			cc.closeFromServerWithErr(err)
			return
		}

		// handle msg
		switch t := msg.(type) {
		case *ReqFilesDataMsg:
			logger.Print("got reqfilesdata")
			msg := &FilesDataMsg{Data: annotatorFilesData}
			if err := cc.write(msg); err != nil {
				cc.closeFromServerWithErr(err)
				return
			}
		case *ReqStartMsg:
			logger.Print("got reqstart")
			cc.ready()
		default:
			// always print if there is a new msg type
			log.Printf("todo: unexpected msg type: %T", t)
		}
	}
}

//----------

func (cc *CConn) sendWhenReady(v *LineMsg) {
	cc.runWhenReady(func() {
		cc.send2(v)
	})
}
func (cc *CConn) send2(v *LineMsg) {
	if syncSend {
		cc.lines = append(cc.lines, v)
		cc.sendLines()
	} else {
		cc.sendLinesInChunks(v)
	}
}
func (cc *CConn) sendLines() {
	if len(cc.lines) == 0 {
		return
	}
	if err := cc.write(cc.lines); err != nil {
		cc.closeFromServerWithErr(err)
		return
	}
	cc.lines = cc.lines[:0]
}
func (cc *CConn) sendLinesInChunks(v *LineMsg) {
	cc.chunks.Lock()
	defer cc.chunks.Unlock()

	cc.lines = append(cc.lines, v)
	if len(cc.lines) >= chunkSendNowNMsgs {
		cc.sendLines()
		return
	}

	if cc.chunks.scheduled {
		return
	}
	cc.chunks.scheduled = true

	cc.chunks.sendWait.Add(1)
	go func() {
		defer cc.chunks.sendWait.Done()

		now := time.Now()
		d := time.Second / time.Duration(chunkSendRate)
		sd := cc.chunks.sent.Add(d).Sub(now)
		cc.chunks.sent = now
		//log.Println("sleeping", sd)
		time.Sleep(sd)

		cc.chunks.Lock()
		defer cc.chunks.Unlock()
		cc.chunks.scheduled = false
		cc.sendLines()
	}()
}

//----------

func (cc *CConn) ready() {
	cc.state.Lock()
	defer cc.state.Unlock()
	if !cc.state.started && !cc.state.closed {
		cc.state.started = true
		cc.state.ready.Broadcast()
	}
}
func (cc *CConn) runWhenReady(fn func()) {
	cc.state.Lock()
	defer cc.state.Unlock()
	for !cc.state.started && !cc.state.closed {
		cc.state.ready.Wait()
	}
	if cc.state.started && !cc.state.closed {
		fn()
	}
}

//----------

func (cc *CConn) write(v interface{}) error {
	encoded, err := EncodeMessage(v)
	if err != nil {
		panic(err)
	}
	n, err := cc.conn.Write(encoded)
	if err != nil {
		return err
	}
	if n != len(encoded) {
		logger.Printf("n!=len(encoded): %v %v\n", n, len(encoded))
	}
	return nil
}
