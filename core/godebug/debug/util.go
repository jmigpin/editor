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
	prefix string
	stdout io.Writer
}

func (l *Logger) logf(f string, args ...any) {
	if l.stdout != nil {
		f = l.prefix + f
		fmt.Fprintf(l.stdout, f, args...)
	}
}
func (l *Logger) logError(err error) {
	l.logf("error: %v", err.Error())
}

func (l *Logger) errorf(f string, args ...any) error {
	return l.error(fmt.Errorf(f, args...))
}
func (l *Logger) error(err error) error {
	return fmt.Errorf("%v%w", l.prefix, err)
}

//----------
//----------
//----------

type PrefixWriter struct {
	writer io.Writer
	prefix string
	buf    bytes.Buffer
}

func NewPrefixWriter(writer io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		writer: writer,
		prefix: prefix,
	}
}
func (p *PrefixWriter) Write(data []byte) (int, error) {
	totalWritten := 0
	for {
		line, err := p.buf.ReadBytes('\n')
		if err == io.EOF {
			p.buf.Write(data)
			break
		}
		if _, err := p.writer.Write([]byte(p.prefix)); err != nil {
			return totalWritten, err
		}
		n, err := p.writer.Write(line)
		totalWritten += n
		if err != nil {
			return totalWritten, err
		}
		data = nil
	}
	return totalWritten, nil
}
