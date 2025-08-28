package termemu

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type VTParser struct {
	emit  func(*TermOp)
	state func() error
	rd    *bufio.Reader

	ansiMode bool // vs VT52
}

func NewVTParser(r io.Reader, emit func(*TermOp)) *VTParser {
	p := &VTParser{emit: emit, ansiMode: true}
	p.rd = bufio.NewReader(r)
	p.state = p.stDefault
	return p
}

func (p *VTParser) Run() error {
	for {
		if err := p.state(); err != nil {
			return err
		}
	}
}

//----------

func (p *VTParser) stDefault() error {
	ru, _, err := p.nextRune()
	if err != nil {
		return err
	}
	return p.handleDefault(ru)
}
func (p *VTParser) handleDefault(ru rune) error {
	switch ru {
	case codeESC:
		p.state = p.stEsc
	case codeCSI:
		if p.ansiMode {
			p.state = p.stCSI
		} else {
			p.emitPrintableRun(ru)
		}
	case codeBEL:
		p.emitKind("bell")
	case codeBS:
		p.emitKind("bs")
	case codeTAB:
		p.emitKind("ht")
	case codeLF, codeVT, codeFF:
		p.emitKind("lf")
	case codeCR:
		p.emitKind("cr")
	case codeG0:
		p.emitKind("g0")
	case codeG1:
		p.emitKind("g1")
	case codeCAN, codeSUB: // TODO
	case codeDEL: // TODO
	case codeNUL: // TODO: bash is dumping all these zeros
	default:
		p.emitPrintableRun(ru)
	}
	return nil
}

func (p *VTParser) stEsc() error {
	b, err := p.nextByte()
	if err != nil {
		return err
	}

	p.state = p.stDefault

	switch b {
	case '#': // DEC special graphics
		p.state = p.stSpecialGraphics
	case '(':
		p.state = p.stGraphics0
	case ')':
		p.state = p.stGraphics1

	case '7': // SC
		p.emitKind("sc")
	case '8': // RC
		p.emitKind("rc")

	case '<':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Exit VT52 mode (Enter VT100 mode).
			p.emitCsi('?', []int{2}, 'h')
		}

	case '=': // alternate keypad mode
	case '>': // Exit alternate keypad mode.

	case 'A':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Cursor up.
			p.emitCsi(0, []int{1}, 'A')
		}
	case 'B':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Cursor down.
			p.emitCsi(0, []int{1}, 'B')
		}
	case 'C':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Cursor right.
			p.emitCsi(0, []int{1}, 'C')
		}
	case 'D':
		if p.ansiMode {
			// IND
			p.emitKind("ind")
		} else {
			// Cursor left.
			p.emitCsi(0, []int{1}, 'D')
		}

	case 'E': // NEL
		p.emitKind("nel")

	case 'F':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Enter graphics mode.
			p.emitKind("g1")
		}
	case 'G':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Exit graphics mode.
			p.emitKind("g0")
		}

	case 'H':
		if p.ansiMode {
			// HTS
			p.emitKind("hts")
		} else { // Move the cursor to the home position.
			p.emitCsi(0, []int{1, 1}, 'H')
		}
	case 'I':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Reverse line feed.
			p.emitKind("ri")
		}
	case 'J':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Erase from the cursor to the end of the screen.
			p.emitCsi(0, []int{0}, 'J')
		}
	case 'K':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Erase from the cursor to the end of the line.
			p.emitCsi(0, []int{0}, 'K')
		}
	case 'M': // RI
		p.emitKind("ri")
	case 'Y':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else {
			// Move the cursor to given row and column.
			bs, err := p.nextBytes(2)
			if err != nil {
				return err
			}
			row1, col1 := int(bs[0])-0x20+1, int(bs[1])-0x20+1
			p.emitCsi(0, []int{row1, col1}, 'H')
		}
	case 'Z':
		if p.ansiMode {
			p.handleDefault(rune(b))
		} else { // Identify.
			p.emitKind("vt52Id")
		}

	case '[':
		if p.ansiMode {
			p.state = p.stCSI
		} else {
			p.handleDefault(rune(b))
		}
	case '\\': // ST // TODO: string terminator
	case ']': // OSC (skip payload)
		if p.ansiMode {
			p.state = p.stOSC
		} else {
			p.handleDefault(rune(b))
		}

	case 'c':
		p.emitKind("ris")

	case codeESC:
		p.state = p.stEsc // cancel and start again

	default:
		// unsupported single ESC: ignore
		if p.ansiMode {
			// commented: aptitude fails with this
			//p.state = p.stDefault
			//p.handleDefault(rune(b))
		} else {
		}
		// DEBUG
		p.emit(&TermOp{kind: "unknownEsc", s: fmt.Sprintf("unhandled: esc %q", rune(b))})
	}
	return nil
}

func (p *VTParser) stCSI() error {
	// collect until final
	w := []byte{}
	for {
		b, err := p.nextByte()
		if err != nil {
			return err
		}

		// abort
		switch b {
		case codeCAN, codeSUB:
			p.state = p.stDefault
			return nil
		case codeESC: // TODO: review
			p.state = p.stEsc
			return nil
		}

		if b >= 0x40 && b <= 0x7e {
			//if len(w) == 0 && b == '[' {
			//	p.state = p.stFnKeySeq
			//	return nil
			//}

			p.state = p.stDefault
			return p.parseCSI(w, b)
		}

		w = append(w, b)
	}
}

func (p *VTParser) stOSC() error {
	p.state = p.stDefault

	// osc = operating system commands
	// OSC ... BEL or ST
	for {
		b, err := p.nextByte()
		if err != nil {
			return err
		}
		if b == codeBEL {
			return nil
		}
		if b == codeESC {
			b2, err2 := p.nextByte()
			if err2 != nil {
				return err2
			}
			if b2 == '\\' { // ST: string terminator
				return nil
			}
		}
	}
}

//----------

func (p *VTParser) stSpecialGraphics() error {
	p.state = p.stDefault

	b, err := p.nextByte()
	if err != nil {
		return err
	}
	switch b {
	case '8': // screen Alignment
		p.emitKind("aln")
	}
	return nil
}

//----------

func (p *VTParser) stGraphics0() error { return p.stGraphics("g0") }
func (p *VTParser) stGraphics1() error { return p.stGraphics("g1") }
func (p *VTParser) stGraphics(typ string) error {
	p.state = p.stDefault

	b, err := p.nextByte()
	if err != nil {
		return err
	}
	op := &TermOp{kind: typ}
	switch b {
	case '0':
		// TODO: g0 is special line drawing?
		op.s = "special"
	case 'B':
		op.s = "ascii"
	}
	p.emit(op)
	return nil
}

//----------

//func (p *VTParser) stFnKeySeq() error {
//	defer func() { p.state = p.stDefault }()

//	b, err := p.nextByte() // 'A'=f1, ...
//	if err != nil {
//		return err
//	}

//	p.Emit(&TermOp{kind: "fnkey", s: string(b)})

//	return nil
//}

//----------
//----------

func (p *VTParser) parseCSI(bs []byte, final byte) error {
	op := &TermOp{kind: "csi"}
	op.csi = &TermCsiOp{}
	op.csi.final = final

	if len(bs) > 0 {
		switch u := bs[0]; u {
		case '?', '>', '<':
			op.csi.priv = u
			bs = bs[1:]
		}
	}

	ps, cancel := p.parseCSIParams(bs)
	if cancel {
		return nil
	}

	op.csi.params = ps

	p.emit(op)

	return nil
}
func (p *VTParser) parseCSIParams(bs []byte) (vals []int, cancel bool) {
	v, seen := 0, false
	for _, b := range bs {
		switch {
		case b >= '0' && b <= '9':
			v = v*10 + int(b-'0')
			seen = true
		case b == ';':
			if seen {
				vals = append(vals, v)
				v, seen = 0, false
			} else {
				vals = append(vals, 0)
			}
		default:
			switch b {
			case codeVT:
				// cancel, dont emit op
				return nil, true
			case codeCR, codeBS:
				// handle now and continue
				p.handleDefault(rune(b))
			default:
				// ignore
			}
		}
	}
	if seen {
		vals = append(vals, v)
	}
	return vals, false
}

//----------

func (p *VTParser) nextByte() (byte, error) {
	return p.rd.ReadByte()
}
func (p *VTParser) nextRune() (rune, int, error) {
	return p.rd.ReadRune()
}

func (p *VTParser) nextBytes(n int) ([]byte, error) {
	w := []byte{}
	for range n {
		b, err := p.rd.ReadByte()
		if err != nil {
			return nil, err
		}
		w = append(w, b)
	}
	return w, nil
}

//----------

func (p *VTParser) emitKind(kind string) {
	p.emit(&TermOp{kind: kind})
}
func (p *VTParser) emitCsi(priv byte, params []int, final byte) {
	p.emit(&TermOp{kind: "csi", csi: &TermCsiOp{priv: priv, params: params, final: final}})
}

//----------

func (p *VTParser) emitPrintableRun(ru rune) {
	//// just one (slower)
	//p.emit(&TermOp{kind: "print", s: string(ru)})
	//return

	// printable run (performance)
	buf := &bytes.Buffer{}
	buf.WriteRune(ru)
	for {
		if p.rd.Buffered() == 0 {
			break
		}
		bs, err := p.rd.Peek(1)
		if err != nil {
			break
		}
		b := bs[0]
		if b < 0x20 || b == codeDEL || b == codeESC || b == codeCSI {
			break
		}
		if _, err := p.rd.Discard(1); err != nil {
			// TODO: log fn error
			//fmt.Println("B err discard", err)
		}
		buf.WriteByte(b)
	}
	p.emit(&TermOp{kind: "print", s: buf.String()})
}

//----------
//----------
//----------

type TermOp struct {
	kind string // csi,bell,bs,print,...
	s    string // used at least in "print"
	csi  *TermCsiOp
}

//----------

type TermCsiOp struct {
	priv   byte // 0=none,'?', '>', ...
	params []int
	final  byte
}

func (op *TermCsiOp) isPriv(b byte) bool {
	return op.priv == b
}
func (op *TermCsiOp) paramDef(idx, def int) int {
	v := op.param(idx)
	if v == 0 {
		v = def
	}
	return v
}
func (op *TermCsiOp) param(idx int) int {
	if idx < 0 || idx >= len(op.params) {
		return 0
	}
	return op.params[idx]
}

func (op *TermCsiOp) A() int           { return op.param(0) }
func (op *TermCsiOp) ADef(def int) int { return op.paramDef(0, def) }
func (op *TermCsiOp) B() int           { return op.param(1) }
func (op *TermCsiOp) BDef(def int) int { return op.paramDef(1, def) }

//----------
//----------
//----------

const (
	codeNUL = 0x00
	codeESC = 0x1b
	codeCSI = 0x9b // control sequence introducer

	codeBS  = 0x08 // backspace, \b, 8
	codeTAB = 0x09 // tab, \t, 9
	codeLF  = 0x0a // linefeed/newline, \n, 10
	codeVT  = 0x0b // vertical tab, 11
	codeFF  = 0x0c // formfeed, 12
	codeCR  = 0x0d // carriage return, \r, 13
	codeDEL = 0x7f
	codeBEL = 0x07

	codeXON  = 0x11
	codeXOFF = 0x13

	codeCAN = 0x18
	codeSUB = 0x1a

	codeSO = 0x0e
	codeSI = 0x0f // 15
	codeG1 = codeSO
	codeG0 = codeSI
)
