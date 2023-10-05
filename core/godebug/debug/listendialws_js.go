//go:build js && editorDebugExecSide

package debug

import (
	"context"
	"errors"
	"fmt"
	"time"

	"syscall/js"
)

func init() {
	//listenReg["ws"] = listenWebsocket
	dialReg["ws"] = dialWebsocket2
	dialReg["wss"] = dialWebsocket2
}

//----------

func dialWebsocket2(ctx context.Context, addr Addr, timeout time.Duration) (Conn, error) {
	u := websocketHostToUrl(addr.String())
	ws := js.Global().Get("WebSocket").New(u)

	// easier to deal with then a blob (could use blob?)
	ws.Set("binaryType", "arraybuffer")

	return newWsConn(addr, ws)
}

//----------

type WsConn struct {
	addr   Addr
	ws     js.Value
	readCh chan any
}

func newWsConn(addr Addr, ws js.Value) (*WsConn, error) {
	wsc := &WsConn{addr: addr, ws: ws}
	wsc.readCh = make(chan any, 1)

	openCh := make(chan any, 1)
	openDone := false

	//----------

	jsEvListen(ws, "error", jsFuncOf2(func(args []js.Value) {
		//jsLog(args)
		jsLog(args[0])
		jsErr := args[0]
		message := jsErr.Get("message").String()
		err := errors.New(message)
		if !openDone {
			openCh <- err
		} else {
			wsc.readCh <- err
		}
	}))
	jsEvListen(ws, "open", jsFuncOf2Release(func(args []js.Value) {
		openDone = true
		openCh <- struct{}{}
	}))
	jsEvListen(ws, "message", jsFuncOf2(func(args []js.Value) {
		//jsLog(args[0])
		jsArr := args[0].Get("data")
		b := arrayBufferToBytes(jsArr)
		wsc.readCh <- b
	}))

	//----------

	v := <-openCh
	switch t := v.(type) {
	case error:
		return nil, t
	}

	return wsc, nil
}

//----------

func (wsc *WsConn) Read(b []byte) (int, error) {
	v := <-wsc.readCh
	switch t := v.(type) {
	case error:
		return 0, t
	case []byte:
		n := copy(b, t)
		return n, nil
	default:
		panic("!")
	}
}
func (wsc *WsConn) Write(b []byte) (int, error) {
	jsArr := bytesToJsArray(b)
	wsc.ws.Call("send", jsArr)
	return len(b), nil // TODO, error
}
func (wsc *WsConn) Close() error {
	return fmt.Errorf("TODO: close")
}
func (wsc *WsConn) LocalAddr() Addr {
	//return wsc.addr
	return nil
}
func (wsc *WsConn) RemoteAddr() Addr {
	return wsc.addr
}

//----------

func (wsc *WsConn) SetDeadline(time.Time) error {
	return fmt.Errorf("setdeadline: todo")
}
func (wsc *WsConn) SetReadDeadline(time.Time) error {
	return fmt.Errorf("setreaddeadline: todo")
}
func (wsc *WsConn) SetWriteDeadline(time.Time) error {
	return fmt.Errorf("setwritedeadline: todo")
}

//----------
//----------
//----------

func arrayBufferToBytes(arrBuf js.Value) []byte {
	// js arraybuffer to js array
	jsArr := js.Global().Get("Uint8Array").New(arrBuf)
	// js array to go slice
	b := make([]byte, jsArr.Get("byteLength").Int())
	js.CopyBytesToGo(b, jsArr)
	return b
}
func bytesToJsArray(b []byte) js.Value {
	// go slice to js array
	jsArr := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(jsArr, b)
	return jsArr
}

//----------

// simplifies to not need to return a value
func jsFuncOf2(fn func(args []js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		fn(args)
		return nil
	})
}

func jsFuncOf2Release(fn func(args []js.Value)) js.Func {
	fn2 := js.Func{}
	fn2 = jsFuncOf2(func(args []js.Value) {
		fn(args)
		fn2.Release()
	})
	return fn2
}

//----------

func jsEvListen(v js.Value, evName string, fn js.Func) {
	v.Call("addEventListener", evName, fn)
}

//----------

func jsLog(args ...any) {
	js.Global().Get("console").Call("log", args...)
}
