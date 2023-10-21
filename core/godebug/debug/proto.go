package debug

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"reflect"
	"sync"
	"time"
)

type Proto struct {
	ctx  context.Context // accept/dial, not conn
	addr Addr
	Side ProtoSide

	isServer        bool
	continueServing bool

	cdr *CtxDoneRelease

	c struct {
		sync.Mutex
		cond        *sync.Cond // have pconn
		pconn       *ProtoConn // server client / client
		ln          Listener   // server side only
		readHeaders bool       // editor side only, read flag
	}

	waitEditorRead struct { // wait for read error, but only on editorside
		sync.Mutex
		cond    *sync.Cond
		haveErr bool
	}
}

func NewProto(ctx context.Context, addr Addr, side ProtoSide, isServer bool, continueServing bool) *Proto {
	p := &Proto{ctx: ctx, addr: addr, Side: side, isServer: isServer, continueServing: continueServing}

	p.c.cond = sync.NewCond(&p.c)
	p.waitEditorRead.cond = sync.NewCond(&p.waitEditorRead)

	return p
}

//----------

func (p *Proto) Connect() error {
	p.c.Lock()
	defer p.c.Unlock()

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
	p.c.pconn = pconn
	p.c.cond.Broadcast()

	p.setupWaitEditorRead()

	if p.cdr == nil {
		p.cdr = newCtxDoneRelease(p.ctx, p.onCtxDone)
	}

	return nil
}

//----------

func (p *Proto) connectServer() (*ProtoConn, error) {
	p.c.readHeaders = false

	// auto start listener
	if p.c.ln == nil {
		ln, err := listen(p.ctx, p.addr)
		if err != nil {
			return nil, err
		}
		p.c.ln = ln
	}

	conn, err := p.c.ln.Accept()
	if err != nil {
		return nil, err
	}

	pconn, err := p.Side.InitProto(conn)
	if err != nil {
		return nil, err
	}
	p.c.readHeaders = true
	return pconn, nil
}
func (p *Proto) connectClient() (*ProtoConn, error) {
	conn, err := dialRetry(p.ctx, p.addr)
	if err != nil {
		return nil, err
	}
	return p.Side.InitProto(conn)
}

//----------

func (p *Proto) getConn() (*ProtoConn, error) {
	p.c.Lock()
	defer p.c.Unlock()
	for p.c.pconn == nil {
		if p.ctx.Err() != nil {
			return nil, p.ctx.Err()
		}
		p.c.cond.Wait()
	}
	return p.c.pconn, nil
}

//----------

func (p *Proto) Read(v any) error {
	if err, ok := p.readHeaders(v); ok {
		return err
	}

	pconn, err := p.getConn()
	if err != nil {
		return err
	}

	err = pconn.Read(v)
	if err != nil {

		if p.shouldContinueServing() {
			if err := p.Connect(); err != nil {
				return err
			}
			return p.Read(v)
		}

		p.editorReadDone()
	}
	return err
}

//----------

func (p *Proto) Write(v any) error {
	pconn, err := p.getConn()
	if err != nil {
		return err
	}
	return pconn.Write(v)
}
func (p *Proto) WriteMsg(m *OffsetMsg) error {
	pconn, err := p.getConn()
	if err != nil {
		return err
	}
	return pconn.WriteMsg(m)
}

//----------

func (p *Proto) onCtxDone() {
	p.editorReadDone()
	if err := p.close(); err != nil {
		_ = err // best effort
	}
}
func (p *Proto) close() error {
	p.c.Lock()
	defer p.c.Unlock()

	p.cdr.Release()

	err0 := (error)(nil)

	if !p.shouldContinueServing() {
		if p.c.ln != nil {
			if err := p.c.ln.Close(); err != nil {
				err0 = err
			}
		}
	}

	if p.c.pconn != nil {
		if err := p.c.pconn.Close(); err != nil {
			err0 = err
		}
	}

	return err0
}

func (p *Proto) WaitClose() error {
	p.waitForEditorRead()
	return p.close()
}

//----------

func (p *Proto) shouldContinueServing() bool {
	dontStop := p.isServer && p.continueServing && p.ctx.Err() == nil
	return dontStop
}

//----------

func (p *Proto) readHeaders(v any) (error, bool) {
	es, ok := p.Side.(*ProtoEditorSide)
	if !ok {
		return nil, false
	}

	p.c.Lock()
	defer p.c.Unlock()

	if !p.c.readHeaders {
		return nil, false
	}
	p.c.readHeaders = false
	return p.readHeaders2(v, es), true
}
func (p *Proto) readHeaders2(v any, es *ProtoEditorSide) error {
	switch t := v.(type) {
	case **FilesDataMsg:
		*t = es.FData
		return nil
	case *FilesDataMsg:
		*t = *es.FData // copy, must have instance
		return nil
	case *any:
		*t = es.FData
		return nil
	default:
		return fmt.Errorf("bad type for filesdatamsg: %T", v)
	}

	// commented: works as well
	//rv := reflect.ValueOf(v)
	//if rv.Kind() != reflect.Pointer {
	//	return fmt.Errorf("expecting pointer: %T", v)
	//}
	//rv = rv.Elem()
	//rv2 := reflect.ValueOf(es.FData)
	//if !rv2.Type().AssignableTo(rv.Type()) {
	//	return fmt.Errorf("%v not assignable to %v", rv2.Type(), rv.Type())
	//}
	//rv.Set(rv2)
	//return nil
}

//----------

// setup to be able to wait for read before closing
func (p *Proto) setupWaitEditorRead() {
	if _, ok := p.Side.(*ProtoEditorSide); !ok {
		return
	}
	p.waitEditorRead.Lock()
	defer p.waitEditorRead.Unlock()
	p.waitEditorRead.haveErr = false
}
func (p *Proto) editorReadDone() {
	if _, ok := p.Side.(*ProtoEditorSide); !ok {
		return
	}
	p.waitEditorRead.Lock()
	defer p.waitEditorRead.Unlock()
	p.waitEditorRead.haveErr = true
	p.waitEditorRead.cond.Broadcast()
}
func (p *Proto) waitForEditorRead() {
	if _, ok := p.Side.(*ProtoEditorSide); !ok {
		return
	}
	p.waitEditorRead.Lock()
	defer p.waitEditorRead.Unlock()
	for !p.waitEditorRead.haveErr {
		p.waitEditorRead.cond.Wait()
	}
}
func (p *Proto) isEditorSide() bool {
	_, ok := p.Side.(*ProtoEditorSide)
	return ok
}

//----------

func (p *Proto) GotConnectedFastCheck() bool {
	// no lock, just a quick check
	// TODO: improve - just used in exec side?
	return p.c.pconn != nil
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
	cdr.release = make(chan struct{}, 1)
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

	logOn bool
}

func (eds *ProtoEditorSide) InitProto(conn Conn) (*ProtoConn, error) {
	pconn := newProtoConn(conn, true)
	if eds.logOn {
		pconn.logOn = true
		pconn.logPrefix = "1:"
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

	logOn bool
}

func (exs *ProtoExecSide) InitProto(conn Conn) (*ProtoConn, error) {
	// allow garbage collect?
	defer func() { exs.fdata = nil }()

	pconn := newProtoConn(conn, !exs.NoWriteBuffering)
	if exs.logOn {
		pconn.logOn = true
		pconn.logPrefix = "1:"
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

	logOn     bool
	logPrefix string
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
	return decode(pconn.conn, v, pconn.logOn, pconn.logPrefix)
}
func (pconn *ProtoConn) Write(v any) error {
	pconn.w.Lock()
	defer pconn.w.Unlock()
	pconn.w.buf.Reset()
	if err := encode(pconn.w.buf, v, pconn.logOn, pconn.logPrefix); err != nil {
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
func (pconn *ProtoConn) Close() error {
	err0 := (error)(nil)
	firstErr := func(err error) {
		if err0 == nil {
			err0 = err
		}
	}

	if pconn.mwb != nil {
		firstErr(pconn.mwb.noMoreWritesAndWait())
	}

	firstErr(pconn.conn.Close())

	return err0
}

//----------
//----------
//----------

type MsgWriteBuffering struct {
	pconn         *ProtoConn
	flushInterval time.Duration

	md struct { // mutex data
		sync.Mutex
		omBuf OffsetMsgs // buffer

		flushing           bool // ex: flushing later
		flushTimer         *time.Timer
		firstFlushWriteErr error
		flushingDone       *sync.Cond
		lastFlush          time.Time
		noMoreWrites       bool
	}
}

func newMsgWriteBuffering(pconn *ProtoConn) *MsgWriteBuffering {
	wb := &MsgWriteBuffering{pconn: pconn}
	wb.flushInterval = time.Second / 10 // minimum times per sec, can be updated more often if the buffer is getting full
	wb.md.omBuf = make([]*OffsetMsg, 0, 4*1024)
	wb.md.flushingDone = sync.NewCond(&wb.md)
	return wb
}
func (wb *MsgWriteBuffering) Write(lm *OffsetMsg) error {
	wb.md.Lock()
	defer wb.md.Unlock()

	if wb.md.noMoreWrites {
		return fmt.Errorf("no more writes allowed")
	}

	// wait for space in the buffer
	for len(wb.md.omBuf) == cap(wb.md.omBuf) { // must be flushing
		if !wb.md.flushing {
			return fmt.Errorf("buffer is full and not flushing")
		}

		// force earlier flush due to buffer being full
		if wb.md.flushTimer.Stop() { // able to stop
			wb.flush()
		}

		if err := wb.waitForFlushingDone(); err != nil {
			return err
		}
	}
	// add to buffer
	wb.md.omBuf = append(wb.md.omBuf, lm)

	if wb.md.flushing { // already added, will be delivered by async flush
		return nil
	}

	wb.md.flushing = true

	now := time.Now()
	deadline := wb.md.lastFlush.Add(wb.flushInterval)

	// commented: always try to buffer (performance)
	// don't async, flush now if already passed the time
	//if now.After(deadline) {
	//	wb.flush()
	// 	return wb.md.flushErr
	//}

	// flush later
	wb.md.flushTimer = time.AfterFunc(deadline.Sub(now), func() {
		wb.md.Lock()
		defer wb.md.Unlock()
		wb.flush()
	})
	return nil
}
func (wb *MsgWriteBuffering) flush() {
	// alway run, even on error
	defer func() {
		wb.md.flushing = false
		wb.md.flushingDone.Broadcast()
	}()
	now := time.Now()
	if err := wb.pconn.Write(&wb.md.omBuf); err != nil {
		if wb.md.firstFlushWriteErr == nil {
			wb.md.firstFlushWriteErr = err
		}
		return
	}
	wb.md.omBuf = wb.md.omBuf[:0]
	wb.md.lastFlush = now
}
func (wb *MsgWriteBuffering) waitForFlushingDone() error {
	for wb.md.flushing {
		wb.md.flushingDone.Wait()
	}
	return wb.md.firstFlushWriteErr
}

//----------

func (wb *MsgWriteBuffering) noMoreWritesAndWait() error {
	wb.md.Lock()
	defer wb.md.Unlock()

	if err := wb.waitForFlushingDone(); err != nil {
		return err
	}

	wb.md.noMoreWrites = true

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
