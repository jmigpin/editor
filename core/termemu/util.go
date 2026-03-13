package termemu

const SeqEsc = "\x1b"
const SeqEscCsi = SeqEsc + "["
const SeqEscO = SeqEsc + "O"

//----------
//----------
//----------

const (
	ModeOff   Mode = iota // no VT emulation
	ModeRaw               // VT emu for replies; present raw bytes
	ModePlain             // VT emu; present printable runes only
	ModeGrid              // VT emu; present rendered text grid
)

//----------

type Mode int

func (m Mode) On() bool {
	return m != ModeOff
}

func (m *Mode) SetBool(m2 Mode, v bool) {
	if *m == m2 && !v {
		*m = ModeOff
		return
	} else if *m != m2 && v {
		*m = m2
	}
}

//----------
//----------
//----------

//func appendRune(b []byte, r rune) []byte {
//	buf := [utf8.UTFMax]byte{}
//	n := utf8.EncodeRune(buf[:], r)
//	return append(b, buf[:n]...)
//}

func appendRune(b []byte, r rune) []byte {
	return append(b, []byte(string(r))...)
}
