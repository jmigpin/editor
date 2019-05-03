package lsproto

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//----------

// Implements rpc.ClientCodec
type JsonCodec struct {
	OnNotificationMessage func(*NotificationMessage)

	rwc       io.ReadWriteCloser
	responses chan interface{}

	// used by read response header/body
	readData readData
}

func NewClientCodec(conn io.ReadWriteCloser) *JsonCodec {
	c := &JsonCodec{rwc: conn}
	c.responses = make(chan interface{}, 1)
	go c.readLoop()
	return c
}

//----------

func (c *JsonCodec) Close() error {
	return c.rwc.Close()
}

//----------

func noreplyMethod(method string) (string, bool) {
	if method[0] == '@' {
		return method[1:], true
	}
	return method, false
}

//----------

func (c *JsonCodec) WriteRequest(req *rpc.Request, data interface{}) error {
	method := req.ServiceMethod

	// methods prefixed with "@" will be considered as not expecting a reply
	noreply := false
	if m, ok := noreplyMethod(method); ok {
		noreply = true
		method = m
	}

	msg := &RequestMessage{
		JsonRpc: "2.0",
		Id:      int(req.Seq),
		Method:  method,
		Params:  data,
	}
	b, err := encodeJson(msg)
	if err != nil {
		return err
	}

	// build header+body
	h := fmt.Sprintf("Content-Length: %v\r\n\r\n", len(b))
	buf := make([]byte, len(h)+len(b))
	copy(buf, []byte(h))  // header
	copy(buf[len(h):], b) // body

	// DEBUG
	logger.Printf("write req -->: %v(%v)", msg.Method, msg.Id)
	//logJson(msg)

	_, err = c.rwc.Write(buf)

	// simulate a response (noreply) with the seq if there is no err writing the msg
	if noreply && err == nil {
		c.responses <- req.Seq
	}

	return errors.Wrap(err, "codec: write req")
}

//----------

func (c *JsonCodec) readLoop() {
	defer func() { close(c.responses) }()
	for {
		cl, err := c.readHeaders() // content-length header
		if err != nil {
			c.responses <- err
			break
		}
		b, err := c.readContent(cl)
		if err != nil {
			c.responses <- err
			break
		}

		// DEBUG
		//logger.Printf("response bytes:\n%s\n", string(b))

		c.responses <- b
	}
}

//----------

func (c *JsonCodec) ReadResponseHeader(resp *rpc.Response) error {
	c.readData = readData{} // reset

	v, ok := <-c.responses
	if !ok {
		return fmt.Errorf("responses chan closed")
	}

	switch t := v.(type) {
	case error:
		return t
	case uint64: // request id that expects noreply
		c.readData = readData{noReply: true}
		resp.Seq = t
		return nil
	case []byte:
		// decode json
		lspResp := &Response{}
		rd := bytes.NewReader(t)
		if err := decodeJson(rd, lspResp); err != nil {
			// an error here will break the client loop, be tolerant
			c.readData = readData{noReply: true}
			log.Printf("decode json error: %v", err)
			return nil
		}
		c.readData.lspResp = lspResp
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
	if c.readData.lspResp.isServerPush() {
		// data should be nil
		if reply != nil {
			panic(fmt.Sprintf("server push with reply expecting data: %v", reply))
		}
		// run callback
		nm := c.readData.lspResp.NotificationMessage
		if c.OnNotificationMessage != nil {
			c.OnNotificationMessage(&nm)
		}
		return nil
	}

	// assign data
	if lspResp, ok := reply.(*Response); ok {
		*lspResp = *c.readData.lspResp
		return nil
	}

	return fmt.Errorf("reply data not assigned")
}

//----------

func (c *JsonCodec) readHeaders() (int, error) {
	var length int
	for {
		// read line
		var line string
		for {
			p := make([]byte, 1)
			_, err := c.rwc.Read(p)
			if err != nil {
				return 0, err
			}
			line += string(p)
			if p[0] == '\n' {
				break
			}
			if len(line) > 1024 {
				return 0, errors.New("header line too long")
			}
		}
		line = strings.TrimSpace(line)
		// header finished
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
	if err != nil {
		return nil, err
	}
	return b, nil
}

//----------

type readData struct {
	noReply bool
	lspResp *Response
}

//----------
