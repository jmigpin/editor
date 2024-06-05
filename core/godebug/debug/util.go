package debug

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

//----------
//----------
//----------

type FnOnCtxDone struct {
	ctx  context.Context
	fn   func()
	once sync.Once
	ch   chan struct{}
}

func NewFnOnCtxDone(ctx context.Context, fn func()) *FnOnCtxDone {
	c := &FnOnCtxDone{ctx: ctx, fn: fn}
	c.ch = make(chan struct{}, 1)
	go func() {
		select {
		case <-c.ch:
		case <-ctx.Done(): // ctx.Err()!=nil if done
			fn()
		}
	}()
	return c
}
func (c *FnOnCtxDone) Cancel() {
	c.once.Do(func() {
		close(c.ch)
	})
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
	l.logf("error: %v", err.Error())
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
