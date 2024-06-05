package debug

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type Proto interface {
	Read(any) error
	Write(any) error
	WriteMsg(*OffsetMsg) error

	// ex: execside(server/client): close after finished sending
	// ex: editorside(server/client): wait for EOF
	CloseOrWait() error
}

func NewProto(ctx context.Context, addr Addr, side ProtoSide, isServer, continueServing bool, stdout io.Writer) (Proto, error) {

	// support connect timeout
	connectCtx := ctx
	if dur, ok := ctx.Value("connectTimeout").(time.Duration); ok {
		ctx3, cancel := context.WithTimeout(ctx, dur)
		defer cancel()
		connectCtx = ctx3
	}

	if isServer {
		return newProtoServer(ctx, connectCtx, addr, side, continueServing, stdout)
	} else {
		return newProtoClient(ctx, connectCtx, addr, side, stdout)
	}
}

//----------
//----------
//----------

type ProtoServer struct {
	ctx             context.Context
	side            ProtoSide
	continueServing bool

	state struct {
		sync.Mutex
		*sync.Cond

		listening  bool
		accepting  bool
		haveConn   bool
		havePconn  bool
		closing    bool
		closingErr error

		conn  Conn
		pconn *ProtoConn
	}

	ln struct {
		ln       Listener
		closeErr error
	}

	ctxStop func() bool

	Logger
}

func newProtoServer(ctx, connectCtx context.Context, addr Addr, side ProtoSide, continueServing bool, stdout io.Writer) (*ProtoServer, error) {
	p := &ProtoServer{
		ctx:             ctx,
		side:            side,
		continueServing: continueServing,
	}
	p.state.Cond = sync.NewCond(&p.state)
	p.Logger.stdout = stdout

	// allow general ctx to stop
	p.ctxStop = context.AfterFunc(ctx, func() {
		_ = p.closeWithCause(ctx.Err())
	})

	if err := p.startListening(connectCtx, addr); err != nil {
		_ = p.closeWithCause(err)
		return nil, err
	}

	return p, nil
}

//----------

func (p *ProtoServer) stateLB(fn func()) {
	p.state.Lock()
	defer p.state.Unlock()
	defer p.state.Broadcast()
	fn()
}

//----------

func (p *ProtoServer) startListening(ctx context.Context, addr Addr) error {
	ln, err := listen(ctx, addr)
	if err != nil {
		return err
	}
	p.ln.ln = ln
	p.stateLB(func() { p.state.listening = true })
	p.logf("listening: %v\n", addr)

	// allow ctx to stop
	stop := context.AfterFunc(ctx, func() {
		_ = p.closeWithCause(ctx.Err())
	})
	defer stop()

	go p.acceptLoop()

	return p.waitForConn(ctx)
}
func (p *ProtoServer) closeListener() error {
	p.stateLB(func() {
		if p.state.listening {
			p.ln.closeErr = p.ln.ln.Close()
			p.state.listening = false
		}
	})
	return p.ln.closeErr
}

//----------

func (p *ProtoServer) acceptLoop() {
	defer p.closeListener()

	p.stateLB(func() { p.state.accepting = true })
	defer p.stateLB(func() { p.state.accepting = false })

	for {
		conn, err := p.ln.ln.Accept()
		if err != nil {
			p.logError(err)
			break
		}

		p.handleAccepted(conn)

		// only one connection
		if !p.continueServing {
			break
		}
	}
}
func (p *ProtoServer) handleAccepted(conn Conn) {
	p.stateLB(func() {
		// don't continue if already/still have a connection
		if p.state.haveConn {
			_ = conn.Close()
			err := fmt.Errorf("already have a connection")
			p.logError(err)
			return
		}

		p.state.conn = conn
		p.state.haveConn = true
	})
}

//----------

func (p *ProtoServer) waitForConn(ctx context.Context) error {
	_, err := p.getPconn(ctx)
	return err
}
func (p *ProtoServer) getPconn(ctx context.Context) (*ProtoConn, error) {
	p.state.Lock()
	defer p.state.Unlock()

	if p.state.havePconn {
		return p.state.pconn, nil
	}

	defer p.state.Broadcast()

	// wait for connection
	for ; p.state.haveConn == false; p.state.Wait() {
		if p.state.closing {
			return nil, p.state.closingErr
		}
	}

	// transition conn to pconn
	conn := p.state.conn
	p.state.conn = nil
	p.state.haveConn = false

	// init proto connection
	pconn, err := InitProtoSide(ctx, p.side, conn, p.stdout)
	if err != nil {
		return nil, err
	}

	p.state.pconn = pconn
	p.state.havePconn = true

	return pconn, nil
}
func (p *ProtoServer) closePconn() error {
	err := error(nil)
	p.stateLB(func() {
		if p.state.havePconn {
			err = p.state.pconn.Close()
			p.state.havePconn = false
		}
	})
	return err
}

//----------

func (p *ProtoServer) Read(v any) error {
	if eds, ok := p.side.(*ProtoEditorSide); ok {
		if ok2, err := eds.readHeaders(v); err != nil || ok2 {
			return err
		}
	}

	pconn, err := p.getPconn(p.ctx)
	if err != nil {
		return err
	}
	return p.monitorPconnErr(pconn.Read(v))
}
func (p *ProtoServer) Write(v any) error {
	pconn, err := p.getPconn(p.ctx)
	if err != nil {
		return err
	}
	return p.monitorPconnErr(pconn.Write(v))
}
func (p *ProtoServer) WriteMsg(m *OffsetMsg) error {
	pconn, err := p.getPconn(p.ctx)
	if err != nil {
		return err
	}
	return p.monitorPconnErr(pconn.WriteMsg(m))
}

//----------

func (p *ProtoServer) monitorPconnErr(err error) error {
	if err != nil {
		p.stateLB(func() {
			p.state.havePconn = false
			if p.state.accepting {
				//err = errors.Join(err, ContinueServingErr)
				err = fmt.Errorf("%w, %w", err, ContinueServingErr)
			}
		})
	}
	return err
}

//----------

func (p *ProtoServer) closeWithCause(err error) error {
	p.stateLB(func() {
		if p.state.closing == false {
			p.state.closing = true
			p.state.closingErr = err
		}
	})

	err1 := p.closeListener()
	err2 := p.closePconn()
	p.ctxStop() // clear resource

	return errors.Join(err1, err2)
}

//----------

func (p *ProtoServer) wait() error {
	p.state.Lock()
	defer p.state.Unlock()
	for p.state.listening ||
		p.state.accepting ||
		p.state.haveConn ||
		p.state.havePconn {
		p.state.Wait()
	}
	return nil
}

//----------

func (p *ProtoServer) CloseOrWait() error {
	switch p.side.(type) {
	case *ProtoEditorSide:
		return p.wait()
	case *ProtoExecSide:
		return p.closeWithCause(errors.New("close"))
	default:
		panic("bad type")
	}
}

//----------
//----------
//----------

type ProtoClient struct {
	ctx  context.Context
	side ProtoSide

	state struct {
		sync.Mutex
		*sync.Cond

		havePconn  bool
		closing    bool
		closingErr error
	}

	pconn *ProtoConn

	ctxStop func() bool

	Logger
}

func newProtoClient(ctx, connectCtx context.Context, addr Addr, side ProtoSide, stdout io.Writer) (*ProtoClient, error) {
	p := &ProtoClient{ctx: ctx, side: side}
	p.state.Cond = sync.NewCond(&p.state)
	p.Logger.stdout = stdout

	// allow general ctx to stop
	p.ctxStop = context.AfterFunc(ctx, func() {
		_ = p.closeWithCause(ctx.Err())
	})

	conn, err := dialRetry(connectCtx, addr)
	if err != nil {
		_ = p.closeWithCause(err)
		return nil, err
	}
	pconn, err := InitProtoSide(connectCtx, side, conn, stdout)
	if err != nil {
		_ = p.closeWithCause(err)
		return nil, err
	}
	p.pconn = pconn
	p.stateLB(func() { p.state.havePconn = true })
	p.logf("connected: %v\n", addr)

	return p, nil
}

//----------

func (p *ProtoClient) stateLB(fn func()) {
	p.state.Lock()
	defer p.state.Unlock()
	defer p.state.Broadcast()
	fn()
}

//----------

func (p *ProtoClient) Read(v any) error {
	if eds, ok := p.side.(*ProtoEditorSide); ok {
		if ok2, err := eds.readHeaders(v); err != nil || ok2 {
			return err
		}
	}

	return p.monitorPconnErr(p.pconn.Read(v))
}

func (p *ProtoClient) Write(v any) error {
	return p.monitorPconnErr(p.pconn.Write(v))
}
func (p *ProtoClient) WriteMsg(m *OffsetMsg) error {
	return p.monitorPconnErr(p.pconn.WriteMsg(m))
}

//----------

func (p *ProtoClient) monitorPconnErr(err error) error {
	if err != nil {
		err = p.closeWithCause(err)
	}
	return err
}

//----------

func (p *ProtoClient) closeWithCause(err error) error {
	err2 := error(nil)
	p.stateLB(func() {
		if p.state.closing == false {
			p.state.closing = true
			p.state.closingErr = err

			if p.state.havePconn {
				p.state.havePconn = false
				err2 = p.pconn.Close()
			}
		}
	})

	p.ctxStop() // clear resource

	return err2
}

//----------

func (p *ProtoClient) wait() error {
	p.state.Lock()
	defer p.state.Unlock()
	for p.state.havePconn {
		p.state.Wait()
	}
	return nil
}

//----------

func (p *ProtoClient) CloseOrWait() error {
	switch p.side.(type) {
	case *ProtoEditorSide:
		return p.wait()
	case *ProtoExecSide:
		return p.closeWithCause(errors.New("close"))
	default:
		panic("bad type")
	}
}

//----------
//----------
//----------

type ProtoSide interface {
	initProto(_ Conn, logStdout io.Writer) (*ProtoConn, error)
}

func InitProtoSide(ctx context.Context, side ProtoSide, conn Conn, logStdout io.Writer) (*ProtoConn, error) {
	// allow ctx to stop initproto() by closing the connection
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stop()

	pconn, err := side.initProto(conn, logStdout)
	if err != nil {
		_ = conn.Close()
		if ctx.Err() != nil {
			err = fmt.Errorf("initprotoside: %w: %w", ctx.Err(), err)
		}
		return nil, err
	}
	return pconn, nil
}

//----------
//----------
//----------

// debug protocol editor side
// 1. send request for files data
// 2. receive files data
// 3. send request for start

type ProtoEditorSide struct {
	pconn   *ProtoConn
	FData   *FilesDataMsg // received from exec side
	fdataMu sync.Mutex
}

func (eds *ProtoEditorSide) initProto(conn Conn, logStdout io.Writer) (*ProtoConn, error) {
	pconn := newProtoConn(conn, true)
	pconn.stdout = logStdout
	pconn.prefix = "1:"

	if err := pconn.Read(&HandshakeMsg{}); err != nil {
		return nil, err
	}

	if err := pconn.Write(&ReqFilesDataMsg{}); err != nil {
		return nil, err
	}
	if err := pconn.Read(&eds.FData); err != nil {
		return nil, err
	}
	if err := pconn.Write(&ReqStartMsg{}); err != nil {
		return nil, err
	}
	return pconn, nil
}

func (eds *ProtoEditorSide) readHeaders(v any) (bool, error) {
	// fast track
	if eds.FData == nil {
		return false, nil
	}

	eds.fdataMu.Lock()
	defer eds.fdataMu.Unlock()

	if eds.FData == nil {
		return false, nil
	}
	defer func() { eds.FData = nil }()

	switch t := v.(type) {
	//case **FilesDataMsg:
	//	*t = eds.FData
	//	return nil
	//case *FilesDataMsg:
	//	*t = *eds.FData // copy, must have instance
	//	return nil
	case *any:
		*t = eds.FData
		return true, nil
	default:
		return false, fmt.Errorf("readheaders: unhandled type: %T", v)
	}
}

//----------
//----------
//----------

// debug protocol executable side
// 1. receive request for files data
// 2. send files data
// 3. receive request for start

type ProtoExecSide struct {
	pconn            *ProtoConn
	fdata            *FilesDataMsg // to be sent, can be discarded
	NoWriteBuffering bool
}

func (exs *ProtoExecSide) initProto(conn Conn, logStdout io.Writer) (*ProtoConn, error) {
	// allow garbage collect
	defer func() { exs.fdata = nil }()

	pconn := newProtoConn(conn, !exs.NoWriteBuffering)
	pconn.stdout = logStdout
	pconn.prefix = "2:"

	// exec side needs to send handshake first, to allow the editor to peek the client intention, the msg should have at least the size of the peek at flexlistener (~10)
	if err := pconn.Write(&HandshakeMsg{Msg: "0000000000"}); err != nil {
		return nil, err
	}

	if err := pconn.Read(&ReqFilesDataMsg{}); err != nil {
		return nil, err
	}
	if err := pconn.Write(exs.fdata); err != nil {
		return nil, err
	}
	if err := pconn.Read(&ReqStartMsg{}); err != nil {
		return nil, err
	}
	return pconn, nil
}

//----------
//----------
//----------

type ProtoConn struct {
	conn Conn

	w struct {
		sync.Mutex
		buf *bytes.Buffer
	}
	r struct {
		sync.Mutex
	}

	mwb *MsgWriteBuffering

	Logger
}

func newProtoConn(conn Conn, mWriteBuffering bool) *ProtoConn {
	pconn := &ProtoConn{conn: conn}
	pconn.w.buf = &bytes.Buffer{}
	if mWriteBuffering {
		pconn.mwb = newMsgWriteBuffering(pconn)
	}
	return pconn
}
func (pconn *ProtoConn) Read(v any) error {
	pconn.r.Lock()
	defer pconn.r.Unlock()
	return decode(pconn.conn, v, pconn.Logger)
}

//----------

func (pconn *ProtoConn) Write(v any) error {
	pconn.w.Lock()
	defer pconn.w.Unlock()
	pconn.w.buf.Reset()
	if err := encode(pconn.w.buf, v, pconn.Logger); err != nil {
		return err
	}
	_, err := pconn.conn.Write(pconn.w.buf.Bytes())
	return err
}
func (pconn *ProtoConn) WriteMsg(m *OffsetMsg) error {
	if pconn.mwb != nil {
		return pconn.mwb.Write(m)
	}
	return pconn.Write(m)
}

//----------

func (pconn *ProtoConn) Close() error {
	w := []error{}
	if pconn.mwb != nil {
		err := pconn.mwb.noMoreWritesAndWait()
		w = append(w, err)
	}
	err := pconn.conn.Close()
	w = append(w, err)
	return errors.Join(w...)
}

//----------
//----------
//----------

type MsgWriteBuffering struct {
	pconn         *ProtoConn
	flushInterval time.Duration

	mu struct { // mutex data
		sync.Mutex
		*sync.Cond

		msgBuf             OffsetMsgs // buffer
		flushing           bool       // ex: flushing later
		flushTimer         *time.Timer
		lastFlush          time.Time
		firstFlushWriteErr error
		noMoreWrites       bool
	}
}

func newMsgWriteBuffering(pconn *ProtoConn) *MsgWriteBuffering {
	wb := &MsgWriteBuffering{pconn: pconn}
	wb.flushInterval = time.Second / 10 // minimum times per sec, can be updated more often if the buffer is getting full
	wb.mu.msgBuf = make([]*OffsetMsg, 0, 4*1024)
	wb.mu.Cond = sync.NewCond(&wb.mu)
	return wb
}
func (wb *MsgWriteBuffering) Write(lm *OffsetMsg) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if wb.mu.noMoreWrites {
		return fmt.Errorf("no more writes allowed")
	}

	// wait for space in the buffer
	for len(wb.mu.msgBuf) == cap(wb.mu.msgBuf) { // must be flushing
		if !wb.mu.flushing {
			return fmt.Errorf("buffer is full and not flushing")
		}

		// force earlier flush due to buffer being full
		if wb.mu.flushTimer.Stop() { // able to stop
			wb.flush()
		}

		if err := wb.waitForFlushingDone(); err != nil {
			return err
		}
	}
	// add to buffer
	wb.mu.msgBuf = append(wb.mu.msgBuf, lm)

	if wb.mu.flushing { // already added, will be delivered by async flush
		return nil
	}
	wb.mu.flushing = true

	now := time.Now()
	deadline := wb.mu.lastFlush.Add(wb.flushInterval)

	// commented: always try to buffer (performance)
	// don't async, flush now if already passed the time
	//if now.After(deadline) {
	//	wb.flush()
	// 	return wb.md.flushErr
	//}

	// flush later
	wb.mu.flushTimer = time.AfterFunc(deadline.Sub(now), func() {
		wb.mu.Lock()
		defer wb.mu.Unlock()
		wb.flush()
	})
	return nil
}
func (wb *MsgWriteBuffering) flush() {
	// always run, even on error
	defer func() {
		wb.mu.flushing = false
		wb.mu.Broadcast()
	}()

	now := time.Now()
	if err := wb.pconn.Write(&wb.mu.msgBuf); err != nil {
		if wb.mu.firstFlushWriteErr == nil {
			wb.mu.firstFlushWriteErr = err
		}
		return
	}
	wb.mu.msgBuf = wb.mu.msgBuf[:0]
	wb.mu.lastFlush = now
}
func (wb *MsgWriteBuffering) waitForFlushingDone() error {
	for wb.mu.flushing {
		wb.mu.Wait()
	}
	return wb.mu.firstFlushWriteErr
}

//----------

func (wb *MsgWriteBuffering) noMoreWritesAndWait() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	wb.mu.noMoreWrites = true
	if err := wb.waitForFlushingDone(); err != nil {
		return err
	}
	return nil
}

//----------
//----------
//----------

var ContinueServingErr = errors.New("continue serving")
