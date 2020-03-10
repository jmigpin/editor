package iorw

import (
	"bytes"

	"github.com/jmigpin/editor/util/evreg"
)

// Runs callbacks on operations.
type RWEvents struct {
	ReadWriter
	EvReg evreg.Register
}

func NewRWEvents(rw ReadWriter) *RWEvents {
	return &RWEvents{ReadWriter: rw}
}

//----------

func (rw *RWEvents) Overwrite(i, n int, p []byte) error {
	// pre write event
	ev := &RWEvPreWrite{i, n, p, nil}
	rw.EvReg.RunCallbacks(RWEvIdPreWrite, ev)
	if ev.ReplyErr != nil {
		return ev.ReplyErr
	}

	if err := rw.ReadWriter.Overwrite(i, n, p); err != nil {
		return err
	}

	// write event
	u := &RWEvWrite{i, n, len(p)}
	rw.EvReg.RunCallbacks(RWEvIdWrite, u)

	// write event 2 (contains content changed flag)
	if rw.EvReg.NCallbacks(RWEvIdWrite2) > 0 {
		changed := true
		if n == len(p) {
			b, err := rw.ReadNAtFast(i, n)
			if err == nil {
				if bytes.Equal(b, p) {
					changed = false
				}
			}
		}
		w := &RWEvWrite2{*u, changed}
		rw.EvReg.RunCallbacks(RWEvIdWrite2, w)
	}
	return nil
}

//----------

const (
	RWEvIdWrite    = iota // ev=RWEvWrite
	RWEvIdWrite2          // ev=RWEvWrite2
	RWEvIdPreWrite        // ev=RWEvPreWrite
)

//----------

type RWEvWrite struct {
	Index int
	Dn    int // n deleted bytes
	In    int // n inserted bytes
}

type RWEvWrite2 struct {
	RWEvWrite
	Changed bool
}

type RWEvPreWrite struct {
	Index    int
	N        int
	P        []byte
	ReplyErr error // can be set by any caller to cancel the write
}
