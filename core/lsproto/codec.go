package lsproto

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/rpc"
	"strconv"
	"strings"
	"sync"

	"errors"
)

//----------

// Implements rpc.ClientCodec
type JsonCodec struct {
	OnNotificationMessage   func(*NotificationMessage)
	OnUnexpectedServerReply func(*Response)

	rwc           io.ReadWriteCloser
	responses     chan interface{}
	simulatedResp chan interface{}

	readData readData // used by read response header/body

	mu struct {
		sync.Mutex
		closed bool
	}
}

// Needs a call to ReadLoop() to start reading.
func NewJsonCodec(rwc io.ReadWriteCloser) *JsonCodec {
	c := &JsonCodec{rwc: rwc}
	c.responses = make(chan interface{}, 4)
	c.simulatedResp = make(chan interface{}, 4)
	return c
}

//----------

func (c *JsonCodec) WriteRequest(req *rpc.Request, data interface{}) error {
	method := req.ServiceMethod

	// internal: methods with a noreply prefix don't expect a reply. This was done throught the method name to be able to use net/rpc. This is not part of the lsp protocol.
	noreply := false
	if m, ok := noreplyMethod(method); ok {
		noreply = true
		method = m
	}

	var msg interface{}
	if  noreply {
		// don't send an Id on messages that don't need an acknowledgement
		msg = &NotificationMessage{
			JsonRpc: "2.0",
			Method:  method,
			Params:  data,
		}
	} else {
		// default case includes the Id
		msg = &RequestMessage{
			JsonRpc: "2.0",
			Id:      int(req.Seq),
			Method:  method,
			Params:  data,
		}
	}
	//logPrintf("write req -->: %v(%v)", msg.Method, msg.Id)

	b, err := encodeJson(msg)
	if err != nil {
		return err
	}

	// build header+body
	h := fmt.Sprintf("Content-Length: %v\r\n\r\n", len(b))
	buf := make([]byte, len(h)+len(b))
	copy(buf, []byte(h))  // header
	copy(buf[len(h):], b) // body

	logPrintf("write req -->: %s%s", h, string(b))

	_, err = c.rwc.Write(buf)
	if err != nil {
		return err
	}

	// simulate a response (noreply) with the seq if there is no err writing the msg
	if noreply {
		c.responses <- req.Seq
	}
	return nil
}

//----------

func (c *JsonCodec) ReadLoop() error {
	for {
		b, err := c.read()
		if c.isClosed() {
			return nil // no error if done
		}
		if err != nil {
			logPrintf("read resp err <--: %v\n", err)
			return err
		}
		logPrintf("read resp <--: %s\n", b)
		c.responses <- b
	}
}

//----------

// Sets response.Seq to have ReadResponseBody be called with the correct reply variable. i
func (c *JsonCodec) ReadResponseHeader(resp *rpc.Response) error {
	c.readData = readData{}          // reset
	resp.Seq = uint64(math.MaxInt64) // set to non-existent sequence

	v, ok := <-c.responses
	if !ok {
		return fmt.Errorf("responses chan closed")
	}

	switch t := v.(type) {
	case uint64: // request id that expects noreply
		c.readData = readData{noReply: true}
		resp.Seq = t
		return nil
	case []byte:
		// decode json
		lspResp := &Response{}
		rd := bytes.NewReader(t)
		if err := decodeJson(rd, lspResp); err != nil {
			return fmt.Errorf("jsoncodec: decode: %v", err)
		}
		c.readData.resp = lspResp
		// msg id (needed for the rpc to run the reply to the caller)
		if !lspResp.isServerPush() {
			resp.Seq = uint64(lspResp.Id)
		}
		return nil
	default:
		panic("!")
	}
}

func (c *JsonCodec) ReadResponseBody(reply interface{}) error {
	// exhaust requests with noreply
	if c.readData.noReply {
		return nil
	}

	// server push callback (no id)
	if c.readData.resp.isServerPush() {
		if reply != nil {
			return fmt.Errorf("jsoncodec: server push with reply expecting data: %v", reply)
		}
		// run callback
		nm := c.readData.resp.NotificationMessage
		if c.OnNotificationMessage != nil {
			c.OnNotificationMessage(&nm)
		}
		return nil
	}

	// assign data
	if replyResp, ok := reply.(*Response); ok {
		*replyResp = *c.readData.resp
		return nil
	}

	// error
	if reply == nil {
		// Server returned a reply that was supposed to have no reply.
		// Can happen if a "noreply:" func was called and the msg id was already thrown away because it was a notification (no reply was expected). A server can reply with an error in the case of not supporting that function. Or reply if it does support, but internaly it returns a msg saying that the function did not reply a value.
		c.unexpectedServerReply(c.readData.resp)
		//err := fmt.Errorf("jsoncodec: server msg without handler: %s", spew.Sdump(c.readData))
		// Returning an error would stop the connection.
		return nil
	}
	return fmt.Errorf("jsoncodec: reply data not assigned: %v", reply)
}

//----------

func (c *JsonCodec) read() ([]byte, error) {
	cl, err := c.readContentLengthHeader() // content length
	if err != nil {
		return nil, err
	}
	return c.readContent(cl)
}

func (c *JsonCodec) readContentLengthHeader() (int, error) {
	var length int
	var headersSize int
	for {
		// read line
		var line string
		for {
			b := make([]byte, 1)
			_, err := io.ReadFull(c.rwc, b)
			if err != nil {
				return 0, err
			}

			if b[0] == '\n' { // end of header line
				break
			}
			if len(line) > 1024 {
				return 0, errors.New("header line too long")
			}

			line += string(b)
			headersSize += len(b)
		}

		if headersSize > 10*1024 {
			return 0, errors.New("headers too long")
		}

		// header finished (empty line)
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		// header line
		colon := strings.IndexRune(line, ':')
		if colon < 0 {
			return 0, fmt.Errorf("invalid header line %q", line)
		}
		name := strings.ToLower(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		switch name {
		case "content-length":
			l, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return 0, fmt.Errorf("failed parsing content-length: %v", value)
			}
			if l <= 0 {
				return 0, fmt.Errorf("invalid content-length: %v", l)
			}
			length = int(l)
		}
	}
	if length == 0 {
		return 0, fmt.Errorf("missing content-length")
	}
	return length, nil
}

func (c *JsonCodec) readContent(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := io.ReadFull(c.rwc, b)
	return b, err
}

//----------

func (c *JsonCodec) unexpectedServerReply(resp *Response) {
	if c.OnUnexpectedServerReply != nil {
		c.OnUnexpectedServerReply(resp)
	}
}

//----------

func (c *JsonCodec) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.mu.closed {
		c.mu.closed = true
		return c.rwc.Close()
	}
	return nil
}

func (c *JsonCodec) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mu.closed
}

//----------

type readData struct {
	noReply bool
	resp    *Response
}

//----------

func noreplyMethod(method string) (string, bool) {
	prefix := "noreply:"
	if strings.HasPrefix(method, prefix) {
		return method[len(prefix):], true
	}
	return method, false
}
