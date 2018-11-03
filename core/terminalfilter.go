package core

import (
	"io"
)

type TerminalFilter struct {
	erow *ERow
	w    io.WriteCloser

	esc     bool
	seq     byte
	escaped int

	// https://en.wikipedia.org/wiki/ANSI_escape_code

	// CSI - Control Sequence Introducer
	csi struct {
		param, intermid []byte
		final           byte
	}
}

func NewTerminalFilter(erow *ERow, w io.WriteCloser) *TerminalFilter {
	return &TerminalFilter{erow: erow, w: w}
}

func (w *TerminalFilter) Close() error {
	return w.w.Close()
}

func (w *TerminalFilter) Write(p []byte) (n int, err error) {
	p2 := w.filterSequences(p)
	n, err = w.w.Write(p2)
	n += w.escaped
	return n, err
}

//----------

func (w *TerminalFilter) filterSequences(p []byte) []byte {
	for i := 0; i < len(p); i++ {
		b := p[i]
		if !w.esc {
			if b == 0x1b {
				w.esc = true
				w.seq = 0
			}
		} else {
			//fmt.Printf("%x, %d, %v\n", b, b, string([]byte{b}))
			switch w.seq {
			case 0:
				if b == '[' {
					w.seq = b
					w.resetCSI()
				} else {
					w.esc = false
				}
			case '[':
				if w.badCSI() {
					w.esc = false
					break
				}
				w.readCSI(&p, &i)
			}
		}
	}
	return p
}

//----------

func (w *TerminalFilter) badCSI() bool {
	// TODO: get a safer limit
	return len(w.csi.param)+len(w.csi.intermid) > 30
}

func (w *TerminalFilter) resetCSI() {
	w.csi.param = w.csi.param[:0]
	w.csi.intermid = w.csi.intermid[:0]
	w.csi.final = 0
}

func (w *TerminalFilter) readCSI(p *[]byte, i *int) {
	b := (*p)[*i]
	// param bytes
	if b >= 0x30 && b <= 0x3f {
		w.csi.param = append(w.csi.param, b)
	}
	// intermediary bytes
	if b >= 0x20 && b <= 0x2f {
		w.csi.intermid = append(w.csi.intermid, b)
	}
	// final byte
	if b >= 0x40 && b <= 0x7e {
		w.csi.final = b
		w.interpretCSI(p, i)
		w.esc = false
	}
}

func (w *TerminalFilter) interpretCSI(p *[]byte, i *int) {
	// "J": ED – Erase in Display
	if string(w.csi.final) == "J" {
		// TODO: csi.params
		//if string(w.csi.param) == "3" {

		// clear scren and reset position
		w.erow.Ed.UI.RunOnUIGoRoutine(func() {
			w.erow.Row.TextArea.SetStrClearHistory("")
			w.erow.Row.TextArea.ClearPos()
		})

		// don't output previous bytes
		(*i)++ // 'J'
		w.escaped += *i
		*p = (*p)[*i:]
		*i = 0
	}
}
