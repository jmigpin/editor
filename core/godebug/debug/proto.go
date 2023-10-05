package debug

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"
)

type Proto struct {
	ctx      context.Context // accept/dial, not conn
	isServer bool
	addr     Addr
	Side     ProtoSide

	md struct { // mutex data
		sync.Mutex
		pconn *ProtoConn
		ln    Listener // server side only
	}

	cdr            *CtxDoneRelease
	waitEditorRead struct {
		sync.Mutex
		eofOrCancel bool
		cond        *sync.Cond
	}
}

func NewProto(ctx context.Context, isServer bool, addr Addr, side ProtoSide) *Proto {
	p := &Proto{ctx: ctx, isServer: isServer, addr: addr, Side: side}
	return p
}

//----------

func (p *Proto) Connect() error {
	p.md.Lock()
	defer p.md.Unlock()

	err := (error)(nil)
	pconn := (*ProtoConn)(nil)
	if p.isServer {
		pconn, err = p.connectServer()
	} else {
		pconn, err = p.connectClient()
	}
	if err != nil {
		return err
	}

	p.setupWaitEditorRead()
	p.cdr = newCtxDoneRelease(p.ctx, p.ctxCanceled)
	p.md.pconn = pconn
	return nil
}
func (p *Proto) connectServer() (*ProtoConn, error) {
	// auto start listener
	if p.md.ln == nil {
		ln, err := listen(p.ctx, p.addr)
		if err != nil {
			return nil, err
		}
		p.md.ln = ln
	}

	conn, err := p.md.ln.Accept()
	if err != nil {
		return nil, err
	}
	return p.Side.InitProto(conn)
}
func (p *Proto) connectClient() (*ProtoConn, error) {
	timeout := 4 * time.Second
	conn, err := dialRetry(p.ctx, p.addr, timeout)
	if err != nil {
		return nil, err
	}
	return p.Side.InitProto(conn)
}

//----------

func (p *Proto) GotConnectedFastCheck() bool {
	return p.md.pconn != nil // not need to lock, just a quick check
}

//----------

func (p *Proto) Read(v any) error {
	err := p.md.pconn.Read(v)
	if errors.Is(err, io.EOF) {
		p.setWaitEditorRead()
	}
	return err
}
func (p *Proto) Write(v any) error {
	return p.md.pconn.Write(v)
}
func (p *Proto) WriteLineMsg(lm *LineMsg) error {
	return p.md.pconn.WriteLineMsg(lm)
}

//----------

func (p *Proto) ctxCanceled() {
	p.setWaitEditorRead()
	if err := p.Close(); err != nil {
		_ = err // best effort
	}
}
func (p *Proto) Close() error {
	p.md.Lock()
	defer p.md.Unlock()

	p.cdr.Release()

	err0 := (error)(nil)

	// server side only: listener
	if p.md.ln != nil {
		if err := p.md.ln.Close(); err != nil {
			err0 = err
		}
	}

	if p.md.pconn != nil {
		if err := p.md.pconn.Close(); err != nil {
			err0 = err
		}
	}
	return err0
}

func (p *Proto) WaitClose() error {
	p.runWaitEditorRead()
	return p.Close()
}

//----------

// setup to be able to wait for read to eof before closing
func (p *Proto) setupWaitEditorRead() {
	if _, ok := p.Side.(*ProtoEditorSide); !ok {
		return
	}
	p.waitEditorRead.cond = sync.NewCond(&p.waitEditorRead)
}
func (p *Proto) setWaitEditorRead() {
	if _, ok := p.Side.(*ProtoEditorSide); !ok {
		return
	}
	p.waitEditorRead.Lock()
	p.waitEditorRead.eofOrCancel = true
	p.waitEditorRead.cond.Broadcast()
	p.waitEditorRead.Unlock()
}
func (p *Proto) runWaitEditorRead() {
	if _, ok := p.Side.(*ProtoEditorSide); !ok {
		return
	}
	p.waitEditorRead.Lock()
	for !p.waitEditorRead.eofOrCancel {
		p.waitEditorRead.cond.Wait()
	}
	p.waitEditorRead.Unlock()
}

//----------
//----------
//----------

type CtxDoneRelease struct {
	ctx         context.Context
	fn          func()
	releaseOnce sync.Once
	release     chan struct{}
}

func newCtxDoneRelease(ctx context.Context, fn func()) *CtxDoneRelease {
	cdr := &CtxDoneRelease{ctx: ctx, fn: fn}
	cdr.release = make(chan struct{})
	go func() {
		select {
		case <-cdr.release:
		case <-ctx.Done(): // ctx.Err()!=nil if done
			fn()
		}
	}()
	return cdr
}
func (cdr *CtxDoneRelease) Release() {
	cdr.releaseOnce.Do(func() {
		close(cdr.release)
	})
}

//----------
//----------
//----------

type ProtoSide interface {
	InitProto(Conn) (*ProtoConn, error)
}

//----------
//----------
//----------

// debug protocol editor side
// 1. send request for files data
// 2. receive files data
// 3. send request for start

type ProtoEditorSide struct {
	pconn *ProtoConn
	FData *FilesDataMsg // received from exec side
}

func (eds *ProtoEditorSide) InitProto(conn Conn) (*ProtoConn, error) {
	pconn := newProtoConn(conn, true)
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

func (exs *ProtoExecSide) InitProto(conn Conn) (*ProtoConn, error) {
	defer func() {
		exs.fdata = nil
	}()

	pconn := newProtoConn(conn, !exs.NoWriteBuffering)
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
	enc  *gob.Encoder
	dec  *gob.Decoder
	lmwb *LinesMsgWriteBuffering // line msg write buffer

}

func newProtoConn(conn Conn, writeBuffering bool) *ProtoConn {
	pconn := &ProtoConn{conn: conn}
	pconn.enc = gob.NewEncoder(conn)
	pconn.dec = gob.NewDecoder(conn)
	if writeBuffering {
		pconn.lmwb = newLinesMsgWriteBuffering(pconn)
	}
	return pconn
}
func (pconn *ProtoConn) Read(v any) error {
	return pconn.dec.Decode(v)
}
func (pconn *ProtoConn) Write(v any) error {
	return pconn.enc.Encode(v)
}
func (pconn *ProtoConn) WriteLineMsg(lm *LineMsg) error {
	if pconn.lmwb != nil {
		return pconn.lmwb.Write(lm)
	}
	u := (any)(lm)
	return pconn.Write(&u)
}
func (pconn *ProtoConn) Close() error {
	if pconn.lmwb != nil {
		if err := pconn.lmwb.noMoreWritesAndWait(); err != nil {
			execSidePrintError(err)
		}
	}
	return pconn.conn.Close()
}

//----------
//----------
//----------

type LinesMsgWriteBuffering struct {
	pconn         *ProtoConn
	flushInterval time.Duration

	md struct { // mutex data
		sync.Mutex
		buf []*LineMsg

		flushing     bool // ex: flushing later
		noMoreWrites bool
		lastFlush    time.Time
		flushingDone *sync.Cond
		flushTimer   *time.Timer
	}
}

func newLinesMsgWriteBuffering(pconn *ProtoConn) *LinesMsgWriteBuffering {
	wb := &LinesMsgWriteBuffering{pconn: pconn}
	wb.flushInterval = time.Second / 10 // minimum times per sec, can be updated more often if the buffer is getting full
	wb.md.buf = make([]*LineMsg, 0, 8*1024)
	wb.md.flushingDone = sync.NewCond(&wb.md)
	return wb
}
func (wb *LinesMsgWriteBuffering) Write(lm *LineMsg) error {
	wb.md.Lock()
	defer wb.md.Unlock()

	if wb.md.noMoreWrites {
		return fmt.Errorf("no more write allowed")
	}

	// wait for space in the buffer
	for len(wb.md.buf) == cap(wb.md.buf) { // must be flushing
		if !wb.md.flushing {
			return fmt.Errorf("buffer is full and not flushing")
		}

		// force earlier flush due to buffer being full
		if wb.md.flushTimer.Stop() {
			return wb.flush()
		} else {
			<-wb.md.flushTimer.C // drain channel
		}

		wb.md.flushingDone.Wait()
	}
	// add to buffer
	wb.md.buf = append(wb.md.buf, lm)

	if wb.md.flushing { // already added, will be delivered by async flush
		return nil
	}

	wb.md.flushing = true

	now := time.Now()
	deadline := wb.md.lastFlush.Add(wb.flushInterval)

	// commented: always try to buffer (performance)
	// don't async, flush now if already passed the time
	//if now.After(deadline) {
	//	return wb.flush()
	//}

	// flush later
	wb.md.flushTimer = time.AfterFunc(deadline.Sub(now), func() {
		wb.md.Lock()
		defer wb.md.Unlock()
		if err := wb.flush(); err != nil {
			execSidePrintError(err)
		}
	})
	return nil
}
func (wb *LinesMsgWriteBuffering) flush() error {
	// alway run, even on error
	defer func() {
		wb.md.flushing = false
		wb.md.flushingDone.Broadcast()
	}()

	now := time.Now()

	u := (any)(wb.md.buf)
	if err := wb.pconn.Write(&u); err != nil {
		return err
	}

	wb.md.buf = wb.md.buf[:0]
	wb.md.lastFlush = now
	return nil
}

//----------

func (wb *LinesMsgWriteBuffering) noMoreWritesAndWait() error {
	wb.md.Lock()
	defer wb.md.Unlock()

	wb.md.noMoreWrites = true

	for wb.md.flushing {
		wb.md.flushingDone.Wait()
	}
	return nil
}

//----------
//----------
//----------

func registerForProtoConn(encoderId string, v any) {
	// commented: needs encoderId to avoid name clashes when self debugging
	//gob.Register(v)

	rt := reflect.TypeOf(v)
	name := rt.String() // ex: *debug.ReqFilesDataMsg

	// after: rt = rt.Elem()
	// 	rt.Name() // ex: ReqFilesDataMsg
	// 	rt.PkgPath() // ex: github.com/jmigpin/editor/core/godebug/debug
	// 	rt.PkgPath() // ex: godebugconfig/debug

	s := fmt.Sprintf("%v:%v", encoderId, name)
	gob.RegisterName(s, v)
}
