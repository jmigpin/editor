package core

import (
	"fmt"
	"image"
	"io"

	"github.com/jmigpin/editor/util/uiutil/event"
)

//godebug:annotatefile

// https://en.wikipedia.org/wiki/ANSI_escape_code
// http://ascii-table.com/ansi-escape-sequences.php
// http://ascii-table.com/ansi-escape-sequences-vt-100.php

//----------

// Maintains state through different write calls.
type TerminalFilter struct {
	erow *ERow
	w    io.Writer

	p struct { // parser
		src     []byte
		i       int // src iterator
		stateFn func() bool
		res     []byte // result
	}

	csi    tfCSI
	csi2   tfCSI2
	cursor image.Point
}

//----------

func NewTerminalFilter(erow *ERow, w io.Writer) *TerminalFilter {
	tf := &TerminalFilter{erow: erow, w: w}
	tf.p.stateFn = tf.parseEscape
	return tf
}

//----------

func (tf *TerminalFilter) Write(p []byte) (int, error) {
	pf := tf.filter(p)
	n, err := tf.w.Write(pf)
	if err != nil {
		return 0, err
	}
	if n != len(pf) {
		return 0, fmt.Errorf("terminalfilter: partial write: %v, %v", n, len(pf))
	}
	return len(p), nil
}

//----------

func (tf *TerminalFilter) filter(p []byte) []byte {
	// reset data if previous write was exausted
	if len(tf.p.src) == 0 {
		tf.p.src = nil
		tf.p.i = 0
	}

	// TODO: max buffer to reset state (case of CSI that has no limit)
	// TODO: need to send if no other write is done?
	tf.p.src = append(tf.p.src, p...)

	// state loop
	tf.p.res = nil
	for {
		if !tf.p.stateFn() {
			break
		}
	}
	return tf.p.res
}

//----------

func (tf *TerminalFilter) nextByte() (byte, bool) {
	if tf.p.i >= len(tf.p.src) {
		return 0, false
	}
	u := tf.p.src[tf.p.i]
	tf.p.i++
	return u, true
}

//----------

func (tf *TerminalFilter) advance() {
	tf.p.src = tf.p.src[tf.p.i:]
	tf.p.i = 0
}

func (tf *TerminalFilter) appendStr(s string) {
	tf.p.res = append(tf.p.res, []byte(s)...)
}

func (tf *TerminalFilter) appendAndAdvance() {
	tf.p.res = append(tf.p.res, tf.p.src[:tf.p.i]...)
	tf.advance()
}

//----------

func (tf *TerminalFilter) parseEscape() bool {
	b, ok := tf.nextByte()
	if !ok {
		return false
	}
	switch b {
	case 27: // escape
		tf.p.stateFn = tf.parseC1
	case 14, 15: //shift out/in
		tf.advance() // filtered
	case 13: // carriage return '\r'
		tf.advance() // filtered
	case 8: // backspace '\b'
		tf.advance() // filtered
	default:
		tf.appendAndAdvance()
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
		tf.p.stateFn = tf.parseCSI
		return true
	case ')', '(':
		tf.csi2 = tfCSI2{header: b} // reset data
		tf.p.stateFn = tf.parseCSI2
		return true
	default:
		//if b >= 0x40 && b <= 0x5f {
	}
	// cancel
	tf.appendAndAdvance()
	tf.p.stateFn = tf.parseEscape
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
		tf.appendAndAdvance()
		tf.p.stateFn = tf.parseEscape
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
		tf.appendAndAdvance()
	}
	tf.advance()
	tf.p.stateFn = tf.parseEscape
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
		tf.appendAndAdvance()
		tf.p.stateFn = tf.parseEscape
	}
	return true
}

func (tf *TerminalFilter) interpretCSI2() {
	switch string(tf.csi2.value) {
	default:
		// do nothing: don't output the sequence
	}
	tf.advance()
	tf.p.stateFn = tf.parseEscape
}

//----------

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

func InputToTerminalBytes(ev interface{}) ([]byte, bool) {
	// $man console codes

	switch t := ev.(type) {
	case *event.KeyDown:
		//[]byte(string(t.Rune))
		//m := t.Mods
		//if m.HasAny(event.ModCtrl) {
		//	switch t.Rune {
		//	case '1':
		//	}
		//}

		switch t.KeySym {
		case event.KSymF1:
			w := append([]byte{27}, []byte("[[11~")...)
			return w, true
		case event.KSymF2:
			//w := append([]byte{27}, []byte("[[12~")...)
			w := []byte("\033[[12~")
			//w := []byte("\033OQ")
			return w, true
		}
	}
	return nil, false
}
