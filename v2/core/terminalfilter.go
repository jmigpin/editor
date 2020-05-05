package core

import (
	"fmt"
	"io"
	"unicode/utf8"
)

//godebug:annotatefile

//----------

type TerminalFilter struct {
	tio  TerminalIO
	erow *ERow // can be nil in tests

	// maintains state through write calls
	p struct { // parser
		src      []byte
		start, i int // advance iter
		stateFn  func() error

		csi tfCsi
	}
}

//----------

func NewTerminalFilter(erow *ERow) *TerminalFilter {
	tio := NewERowTermIO(erow)
	return NewTerminalFilter2(tio, erow)
}

func NewTerminalFilter2(tio TerminalIO, erow *ERow) *TerminalFilter {
	tf := &TerminalFilter{tio: tio, erow: erow}
	tf.p.stateFn = tf.stParseDefault
	tf.tio.Init(tf)
	return tf
}

//----------

func (tf *TerminalFilter) Write(p []byte) (int, error) {
	if tf.erow != nil && tf.erow.terminalOpt.filter {
		tf.filter(p)
	} else {
		if err := tf.tio.WriteOp(p); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (tf *TerminalFilter) Read(p []byte) (int, error) {
	return tf.tio.Read(p)
}

func (tf *TerminalFilter) Close() error {
	return tf.tio.Close()
}

//----------

func (tf *TerminalFilter) filter(p []byte) {
	tf.p.src = append(tf.p.src, p...)

	// state loop
	for {
		// from any state, allow escape to cancel previous state
		ru, err := tf.peekRune()
		if err == nil && ru == 0x1b {
			tf.p.stateFn = tf.stParseDefault
		}

		if err := tf.p.stateFn(); err != nil {
			if err != io.EOF {
				// just output error, don't stop the filter (not programs fault)
				if tf.erow != nil {
					tf.erow.Ed.Messagef("terminal filter: error: %v", err)
				}
			}
			break
		}
	}

	// reset data if src was exausted
	discard := len(tf.p.src) > 1024 // safety, don't deal with long esc sequences
	if len(tf.p.src) == tf.p.i || discard {
		tf.p.src = nil
		tf.p.start = 0
		tf.p.i = 0
	}
}

//----------

func (tf *TerminalFilter) nextRune() (rune, error) { return tf.readRune(false) }
func (tf *TerminalFilter) peekRune() (rune, error) { return tf.readRune(true) }

func (tf *TerminalFilter) readRune(peek bool) (rune, error) {
	if tf.p.i >= len(tf.p.src) {
		return 0, io.EOF
	}
	u := tf.p.src[tf.p.i:]
	ru, size := utf8.DecodeRune(u)
	if !peek {
		tf.p.i += size
	}
	return ru, nil
}

func (tf *TerminalFilter) advance() {
	tf.p.start = tf.p.i // advance
}
func (tf *TerminalFilter) value() []byte {
	return tf.p.src[tf.p.start:tf.p.i]
}

//----------

func (tf *TerminalFilter) stParseDefault() error {
	ru, err := tf.nextRune()
	if err != nil {
		return err
	}
	switch ru {
	case 0x1b: // ESC: (1b,27)
		tf.p.stateFn = tf.stParseEsc
	case 0x9b: // CSI, equivalent to "ESC ["
		tf.p.csi = tfCsi{} // reset data
		tf.p.stateFn = tf.stParseCsi
	case 0x7: // BEL (0x7, ^G) beeps
		tf.advance()
	case '\b': // backspace: (8,8), '\b'
		tf.advance()
	case 0x9: // next tab stop or end of the line
	case '\n': // newline: (a,10), '\n'
		tf.advance()
		return tf.tio.WriteOp([]byte{'\n'})
	case 0xb, 0xc: // formfeed, verticaltab
		tf.advance()
	case '\r': // carriage return: (d,13), '\r'
		tf.advance()
	case 0xe: // activate G1 char set: (e,14)
		tf.advance()
	case 0xf: // activate G0 char set: (f,15)
		tf.advance()
	case 0x18, 0x1a: // interrupt escape sequences
		tf.advance()
	case 0x7f: // DEL, ignored
		tf.advance()
	default:
		// not a control byte, add to output
		defer tf.advance()
		return tf.tio.WriteOp(tf.value())
	}
	return nil
}

//----------

func (tf *TerminalFilter) stParseEsc() error {
	ru, err := tf.nextRune()
	if err != nil {
		return err
	}
	switch ru {
	case '[': // CSI
		tf.p.csi = tfCsi{} // reset data
		tf.p.stateFn = tf.stParseCsi
	case ']': // Operating system command
		tf.p.stateFn = tf.stParseOsc
	case '(': // start sequence defining G0 char set
		tf.p.stateFn = tf.stParseG0
	case ')': // start sequence defining G1 char set
		tf.p.stateFn = tf.stParseG1
	case '#': // screen alignment test - fill screen with E's.
		tf.p.stateFn = tf.stParseAlignmentTest

	case 'D': // linefeed
		tf.p.stateFn = tf.stParseDefault
		defer tf.advance()
		return tf.tio.WriteOp([]byte{'\n'})
	case 'E': // newline
		tf.p.stateFn = tf.stParseDefault
		defer tf.advance()
		return tf.tio.WriteOp([]byte{'\n'})
	case 'H': // Set tab stop at current column
	case 'M': // reverse line feed
		tf.advance()
		tf.p.stateFn = tf.stParseDefault
	case '7': // Save cursor
		tf.advance()
		tf.p.stateFn = tf.stParseDefault
	case '8': // Restore cursor location
		tf.advance()
		tf.p.stateFn = tf.stParseDefault
	default:
		tf.todo("stParseEsc: %s", tf.value())
		tf.advance()
		tf.p.stateFn = tf.stParseDefault
	}
	return nil
}

//----------

// csi = Control Sequence Introducer
func (tf *TerminalFilter) stParseCsi() error {
	ru, err := tf.nextRune()
	if err != nil {
		return err
	}

	switch {
	case ru == '?': // 0x3f: params may be preceded by a single question mark.
		tf.p.csi.qMark = true
	case ru == '[':
		// However,  after  CSI [ (or ESC [ [) a single character is read and this entire sequence is ignored.  (The idea is to ignore an echoed  function key.)

		_, _ = tf.nextRune() // TODO: improve
		tf.advance()
		tf.p.stateFn = tf.stParseDefault

	case ru >= 0x30 && ru < 0x3f: // param bytes: 0–9:;<=>
		tf.p.csi.param = append(tf.p.csi.param, ru)
	case ru >= 0x20 && ru <= 0x2f: // intermediary bytes: space !"#$%&'()*+,-./
		tf.p.csi.intermid = append(tf.p.csi.intermid, ru)
	case ru >= 0x40 && ru <= 0x7e: // final byte: @A–Z[\]^_`a–z{|}~
		tf.p.csi.final = ru
		tf.p.stateFn = tf.stParseDefault
		defer tf.advance()
		return tf.interpretCSI()
	default:
		tf.todo("stParseCsi: %s", tf.value())
		tf.advance()
		tf.p.stateFn = tf.stParseDefault
	}
	return nil
}

//----------

func (tf *TerminalFilter) interpretCSI() error {
	switch tf.p.csi.final {
	case 'J': // erase display
		return tf.eraseDisplay()
	}
	return nil
}

func (tf *TerminalFilter) eraseDisplay() error {
	switch string(tf.p.csi.param) {
	case "1": // erase from start to cursor
	case "2": // erase whole display
		return tf.tio.WriteOp("clear")
	case "3": // erase whole display including scroll-back buffer
		return tf.tio.WriteOp("clear")
	default: // from cursor to end of display
	}
	return nil
}

//----------

func (tf *TerminalFilter) stParseOsc() error {
	ru, err := tf.nextRune()
	if err != nil {
		return err
	}
	switch ru {
	case 'P': // set palette // TODO: parse nrrggbb
	case 'R': // reset palette
	default:
		tf.todo("stParseOsc: %s", tf.value())
	}
	return nil
}

//----------

func (tf *TerminalFilter) stParseG0() error { return tf.stParseG(0) }
func (tf *TerminalFilter) stParseG1() error { return tf.stParseG(1) }

func (tf *TerminalFilter) stParseG(g int) error {
	ru, err := tf.nextRune()
	if err != nil {
		return err
	}
	switch ru {
	//case 'B': // Select default (ISO 8859-1 mapping)
	case '0': // Select VT100 graphics mapping
	//case 'U': // Select null mapping - straight to character ROM
	//case 'K': // Select user mapping - the map that is loaded by the utility mapscrn(8).
	default:
		tf.todo("stParseG: %s, g=%v", tf.value(), g)
	}
	tf.advance()
	tf.p.stateFn = tf.stParseDefault
	return nil
}

//----------

func (tf *TerminalFilter) stParseAlignmentTest() error {
	ru, err := tf.nextRune()
	if err != nil {
		return err
	}
	switch ru {
	case '8': // fill screen with E's
	default:
		tf.todo("stParseAlignmentTest: %s", tf.value())
	}
	tf.advance()
	tf.p.stateFn = tf.stParseDefault
	return nil
}

//----------

func (tf *TerminalFilter) todo(f string, a ...interface{}) {
	tf.debug("todo: "+f, a...)
}

func (tf *TerminalFilter) debug(f string, a ...interface{}) {
	f = "tfdebug: " + f
	if tf.erow != nil {
		tf.erow.Ed.Messagef(f, a...)
	} else {
		fmt.Printf(f, a...)
	}
}

//----------

type tfCsi struct {
	param    []rune
	intermid []rune
	final    rune
	qMark    bool
}

//----------
