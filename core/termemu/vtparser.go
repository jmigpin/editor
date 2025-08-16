package termemu

import (
	"bufio"
	"io"
)

type VTParser struct {
	Emit  func(*TermOp)
	state func() error
	rd    *bufio.Reader
}

func NewVTParser(r io.Reader, emit func(*TermOp)) *VTParser {
	p := &VTParser{Emit: emit}
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
	switch ru {
	case 0x1b: // ESC
		p.state = p.stEsc
	case 0x9b: // CSI single
		p.state = p.stCSI
	case 0x07: // BEL
		p.emitKind("bell")
	case '\b':
		p.emitKind("bs")
	case '\n':
		p.emitKind("lf")
	case 0x0b, 0x0c: // VT/FF
	case '\r':
		p.emitKind("cr")
	case 0x0e, 0x0f: // SO/SI (G1/G0)
	case 0x18, 0x1a: // CAN/SUB
	case 0x7f: // DEL
	default:
		// emit rune (direct, slower)
		p.Emit(&TermOp{kind: "print", s: string(ru)})

		//// printable run (performance)
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
	switch b {
	case '[': // CSI
		p.state = p.stCSI
		return nil
	case ']': // OSC (skip payload)
		p.state = p.stOSC
		return nil
	case '\\': // ST
		p.state = p.stDefault
		return nil
	default:
		// unsupported single ESC: ignore
		p.state = p.stDefault
		return nil
	}
}

func (p *VTParser) stCSI() error {
	// collect until final (0x40..0x7E)
	w := []byte{}
	for {
		// TODO: break out after n runes (safe side)?

		b, err := p.nextByte()
		if err != nil {
			return err
		}
		if b >= 0x40 && b <= 0x7E {
			if len(w) == 0 && b == '[' {
				p.state = p.stFnKeySeq
				return nil
			}

			defer func() { p.state = p.stDefault }()
			return p.parseCSI(w, b)
		}
		w = append(w, b)
	}
}

func (p *VTParser) stFnKeySeq() error {
	defer func() { p.state = p.stDefault }()

	b, err := p.nextByte() // 'A'=f1, ...
	if err != nil {
		return err
	}

	p.Emit(&TermOp{kind: "fnkey", s: string(b)})

	return nil
}

func (p *VTParser) stOSC() error {
	defer func() { p.state = p.stDefault }()

	// OSC ... BEL or ST
	for {
		b, err := p.nextByte()
		if err != nil {
			return err
		}
		if b == 0x07 { // BEL
			return nil
		}
		if b == 0x1b {
			nb, err2 := p.nextByte()
			if err2 != nil {
				return err2
			}
			if nb == '\\' {
				return nil
			} // ST
		}
	}
}

//----------

func (p *VTParser) parseCSI(b []byte, final byte) error {
	op := &TermOp{kind: "csi"}
	op.csi.final = final

	if len(b) > 0 {
		switch u := b[0]; u {
		case '?', '>':
			op.csi.hasPriv = true
			op.csi.priv = u
			b = b[1:]
		}
	}

	op.csi.params = p.parseCSIParams(b)

	p.Emit(op)

	return nil
}
func (p *VTParser) parseCSIParams(b []byte) (vals []int) {
	v, seen := 0, false
	for _, c := range b {
		switch {
		case c >= '0' && c <= '9':
			v = v*10 + int(c-'0')
			seen = true
		case c == ';':
			if seen {
				vals = append(vals, v)
				v, seen = 0, false
			} else {
				vals = append(vals, 0)
			}
		}
	}
	if seen {
		vals = append(vals, v)
	}
	return vals
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
	p.Emit(&TermOp{kind: kind})
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
