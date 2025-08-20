package termemu

import (
	"bufio"
	"io"
)

type VTParser struct {
	emit  func(*TermOp)
	state func() error
	rd    *bufio.Reader
}

func NewVTParser(r io.Reader, emit func(*TermOp)) *VTParser {
	p := &VTParser{emit: emit}
	p.state = p.stDefault
	p.rd = bufio.NewReader(r)
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
		p.state = p.stCSI
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

	case codeCAN, codeSUB: // TODO
	case 0x0e, 0x0f: // SO/SI (G1/G0)
	case codeDEL: // TODO

	default:
		// emit rune (direct, slower)
		p.emit(&TermOp{kind: "print", s: string(ru)})

		//// printable run (performance)
		//// TODO: issues with last peek halting the read
		//buf := &bytes.Buffer{}
		//buf.WriteRune(ru)
		//for {
		//	bs, err := p.rd.Peek(1)
		//	if err != nil {
		//		break
		//	}
		//	b := bs[0]
		//	if b < 0x20 || b == 0x7f || b == 0x1b || b == 0x9b {
		//		break
		//	}
		//	if _, err := p.rd.Discard(1); err != nil {
		//		// TODO: log fn error
		//		//fmt.Println("B err discard", err)
		//	}
		//	buf.WriteByte(b)
		//}
		//p.Emit(&TermOp{kind: "print", s: buf.String()})
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
	case codeESC:
		p.state = p.stEsc // cancel and start again
	case '[': // CSI
		p.state = p.stCSI
	case ']': // OSC (skip payload)
		p.state = p.stOSC
	case '\\': // ST // TODO: string terminator
	case 'H': // HTS
		p.emitKind("hts")
	case 'D': // IND
		p.emitKind("ind")
	case 'M': // RI
		p.emitKind("ri")
	case 'E': // NEL
		p.emitKind("nel")
	case '7': // SC
		p.emitKind("sc")
	case '8': // RC
		p.emitKind("rc")
	case '#': // DEC special graphics
		b2, err2 := p.nextByte()
		if err2 != nil {
			return err2
		}
		if b2 == '8' { // screen Alignment
			p.emitKind("aln")
		}
	default:
		// unsupported single ESC: ignore
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

func (p *VTParser) parseCSI(bs []byte, final byte) error {
	op := &TermOp{kind: "csi"}
	op.csi.final = final

	if len(bs) > 0 {
		switch u := bs[0]; u {
		case '?', '>', '<':
			op.csi.hasPriv = true
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

//----------

func (p *VTParser) emitKind(kind string) {
	p.emit(&TermOp{kind: kind})
}

//----------
//----------
//----------

type TermOp struct {
	kind string // csi,bell,bs,print,...
	s    string // used at least in "print"
	csi  struct {
		final   byte
		params  []int
		hasPriv bool
		priv    byte // '?', '>', ...
	}
}

func (op *TermOp) csiPrivIs(b byte) bool {
	return op.csi.hasPriv == true && op.csi.priv == b
}

func (op *TermOp) csiA() int           { return op.csiParam(0) }
func (op *TermOp) csiADef(def int) int { return op.csiParamDef(0, def) }
func (op *TermOp) csiB() int           { return op.csiParam(1) }
func (op *TermOp) csiBDef(def int) int { return op.csiParamDef(1, def) }

//----------

func (op *TermOp) csiParamDef(idx, def int) int {
	v := op.csiParam(idx)
	if v == 0 {
		v = def
	}
	return v
}
func (op *TermOp) csiParam(idx int) int {
	if idx < 0 || idx >= len(op.csi.params) {
		return 0
	}
	return op.csi.params[idx]
}

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
)
