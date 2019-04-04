package core

import (
	"io"
)

// https://en.wikipedia.org/wiki/ANSI_escape_code
// http://ascii-table.com/ansi-escape-sequences.php
// http://ascii-table.com/ansi-escape-sequences-vt-100.php

//----------

// Maintains state through different write calls.
type TerminalFilter struct {
	erow *ERow
	w    io.WriteCloser
	rd   io.ReadCloser

	src   []byte
	srci  int
	dst   []byte
	stf   func() bool // state func
	start int

	csi  tfCSI
	csi2 tfCSI2
}

type tfCSI struct {
	param    []byte
	intermid []byte
	final    byte
}

type tfCSI2 struct {
	header byte
	value  byte
}

//----------

func NewTerminalFilter(erow *ERow, w io.WriteCloser) *TerminalFilter {
	tf := &TerminalFilter{erow: erow, w: w}
	tf.stf = tf.parseEscape
	return tf
}

func NewTerminalFilterReader(erow *ERow, rd io.ReadCloser) *TerminalFilter {
	tf := &TerminalFilter{erow: erow, rd: rd}
	tf.stf = tf.parseEscape
	return tf
}

//----------

func (tf *TerminalFilter) Close() error {
	if tf.rd != nil {
		return tf.rd.Close()
	}
	return tf.w.Close()
}

//----------

func (tf *TerminalFilter) Write(p []byte) (n int, err error) {
	p2 := tf.filter(p)
	n, err = tf.w.Write(p2)
	u := len(p) - (len(p2) - n) // after filtering, report only the bytes not written
	return u, err
}

//----------

func (tf *TerminalFilter) Read(p []byte) (int, error) {
	tmp := make([]byte, len(p))
	n, err := tf.rd.Read(tmp)
	if n > 0 {
		pf := tf.filter(tmp[:n])

		// ensure the filtered result is not bigger then the src
		if len(pf) > len(p) {
			panic("len pf > len p")
		}

		copy(p, pf)
		return len(pf), nil
	}
	return n, err
}

//----------

func (tf *TerminalFilter) filter(p []byte) []byte {
	// reset data if previous write was exausted
	if tf.start == len(tf.src) {
		tf.src = nil
		tf.srci = 0
		tf.start = 0
	}

	// TODO: max buffer to reset state (case of CSI that has no limit)
	// TODO: need to send if no other write is done?
	tf.src = append(tf.src, p...)

	// state loop
	tf.dst = nil
	for {
		if !tf.stf() {
			break
		}
	}
	return tf.dst
}

//----------

func (tf *TerminalFilter) nextByte() (byte, bool) {
	if tf.srci >= len(tf.src) {
		return 0, false
	}
	u := tf.src[tf.srci]
	tf.srci++
	return u, true
}

//----------

func (tf *TerminalFilter) append(b byte) {
	tf.dst = append(tf.dst, b)
	tf.advanceStart()
}

func (tf *TerminalFilter) appendFromStart() {
	for i := tf.start; i < tf.srci; i++ {
		tf.append(tf.src[i])
	}
}

func (tf *TerminalFilter) advanceToParseEscape() {
	tf.stf = tf.parseEscape
	tf.advanceStart()
}

func (tf *TerminalFilter) advanceStart() {
	tf.start = tf.srci
}

//----------

func (tf *TerminalFilter) parseEscape() bool {
	b, ok := tf.nextByte()
	if !ok {
		return false
	}
	switch b {
	case 27: // escape
		tf.stf = tf.parseC1
	case 14, 15: //shift out/in
		tf.advanceStart() // filtered
	case 13: // carriage return
		tf.advanceStart() // filtered
	default:
		tf.append(b)
	}
	return true
}

func (tf *TerminalFilter) parseC1() bool {
	b, ok := tf.nextByte()
	if !ok {
		return false
	}
	switch b {
	case '[':
		tf.csi = tfCSI{} // reset data
		tf.stf = tf.parseCSI
		return true
	case ')', '(':
		tf.csi2 = tfCSI2{header: b} // reset data
		tf.stf = tf.parseCSI2
		return true
	default:
		//if b >= 0x40 && b <= 0x5f {
	}
	// cancel
	tf.appendFromStart()
	tf.advanceToParseEscape()
	return true
}

//----------

// parse Control Sequence Introducer
func (tf *TerminalFilter) parseCSI() bool {
	b, ok := tf.nextByte()
	if !ok {
		return false
	}
	switch {
	case b >= 0x30 && b <= 0x3f: // param bytes: 0–9:;<=>?
		tf.csi.param = append(tf.csi.param, b)
	case b >= 0x20 && b <= 0x2f: // intermediary bytes: space !"#$%&'()*+,-./
		tf.csi.intermid = append(tf.csi.intermid, b)
	case b >= 0x40 && b <= 0x7e: // final byte: @A–Z[\]^_`a–z{|}~
		tf.csi.final = b
		tf.interpretCSI()
	default:
		// cancel
		tf.appendFromStart()
		tf.advanceToParseEscape()
	}
	return true
}

func (tf *TerminalFilter) interpretCSI() {
	switch string(tf.csi.final) {
	case "A": // Cursor up
	case "B": // Cursor down
	case "C": // Cursor forward
	case "D": // Cursor back
	case "H": // Cursor position
	case "J": // Erase in Display
		// TODO: if string(w.csi.param) == "3"

		// clear screen and reset position
		tf.erow.Ed.UI.RunOnUIGoRoutine(func() {
			ta := tf.erow.Row.TextArea
			ta.SetStrClearHistory("")
			ta.ClearPos()
		})
	case "K": // Erase in Line
	case "m": // Select Graphic Rendition

	case "r", "l", "h", "c", "d": // ?
	case "X", "G": // ?

	default:
		tf.appendFromStart()
	}
	tf.advanceToParseEscape()
}

//----------

func (tf *TerminalFilter) parseCSI2() bool {
	b, ok := tf.nextByte()
	if !ok {
		return false
	}
	switch {
	case b >= '0' && b <= '2':
		tf.csi2.value = b
		tf.interpretCSI2()
	default:
		// cancel
		tf.appendFromStart()
		tf.advanceToParseEscape()
	}
	return true
}

func (tf *TerminalFilter) interpretCSI2() {
	switch string(tf.csi2.value) {
	default:
		// do nothing: don't output the sequence
	}
	tf.advanceToParseEscape()
}
