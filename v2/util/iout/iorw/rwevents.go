package iorw

import (
	"github.com/jmigpin/editor/v2/util/evreg"
)

// Runs callbacks on operations.
type RWEvents struct {
	ReadWriterAt
	EvReg evreg.Register
}

func NewRWEvents(rw ReadWriterAt) *RWEvents {
	return &RWEvents{ReadWriterAt: rw}
}

func (rw *RWEvents) OverwriteAt(i, n int, p []byte) error {
	// pre write event
	ev := &RWEvPreWrite{i, n, p, nil}
	rw.EvReg.RunCallbacks(RWEvIdPreWrite, ev)
	if ev.ReplyErr != nil {
		return ev.ReplyErr
	}

	// write event 2 data (contains content changed flag)
	changed := true
	if rw.EvReg.NCallbacks(RWEvIdWrite2) > 0 {
		if eq, err := REqual(rw, i, n, p); err == nil && eq {
			changed = false
		}
	}

	if err := rw.ReadWriterAt.OverwriteAt(i, n, p); err != nil {
		return err
	}

	// write event
	u := &RWEvWrite{i, n, len(p)}
	rw.EvReg.RunCallbacks(RWEvIdWrite, u)

	// write event 2 (contains content changed flag)
	w := &RWEvWrite2{*u, changed}
	rw.EvReg.RunCallbacks(RWEvIdWrite2, w)

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
