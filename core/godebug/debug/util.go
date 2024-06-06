package debug

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
)

// a simple addr implementation
type AddrImpl struct {
	net, str string
}

func NewAddrI(network, str string) *AddrImpl {
	return &AddrImpl{net: network, str: str}
}
func (addr *AddrImpl) Network() string {
	return addr.net
}
func (addr *AddrImpl) String() string {
	h := addr.str
	if k := strings.Index(h, ":"); k == 0 { // ex: ":8080"
		//h = "localhost" + h
		h = "127.0.0.1" + h
	}
	return h
}

//----------
//----------
//----------

type ConnFnCloser struct {
	net.Conn
	closeFn func() error
}

func (c *ConnFnCloser) Close() error {
	return c.closeFn()
}

//----------
//----------
//----------

type Logger struct {
	Prefix string
	W      io.Writer // ex: os.stderr
}

func (l *Logger) logf(f string, args ...any) {
	if l.W != nil {
		f = l.Prefix + f
		fmt.Fprintf(l.W, f, args...)
	}
}
func (l *Logger) logError(err error) {
	l.logf("error: %v\n", err.Error())
}

func (l *Logger) errorf(f string, args ...any) error {
	return l.error(fmt.Errorf(f, args...))
}
func (l *Logger) error(err error) error {
	return fmt.Errorf("%v%w", l.Prefix, err)
}

//----------
//----------
//----------

type PrefixWriter struct {
	writer    io.Writer
	prefix    string
	lineStart bool
}

func NewPrefixWriter(writer io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		writer:    writer,
		prefix:    prefix,
		lineStart: true,
	}
}
func (p *PrefixWriter) Write(data []byte) (int, error) {
	written := 0 // from data slice
	for len(data) > 0 {
		// write prefix
		if p.lineStart {
			p.lineStart = false
			if _, err := p.writer.Write([]byte(p.prefix)); err != nil {
				return written, err
			}
		}
		// find newline
		k := len(data)
		i := bytes.IndexByte(data, '\n')
		if i >= 0 {
			k = i + 1
			p.lineStart = true // next line will be a line start
		}
		// write line
		n, err := p.writer.Write(data[:k])
		written += n
		if err != nil {
			return written, err
		}
		// advance
		data = data[n:]
	}
	return written, nil
}

//----------

const websocketEntryPath = "/editor_debug_ws"

func websocketEntryPathUrl(host string) string {
	return fmt.Sprintf("ws://%s%s", host, websocketEntryPath)
}
