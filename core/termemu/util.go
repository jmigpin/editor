package termemu

import (
	"fmt"
)

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

//func (m *Mode) Set(m2 Mode) error {
//	if *m != ModeOff && *m != m2 {
//		return m.confliftErr(m2)
//	}
//	*m = m2
//	return nil
//}

func (m *Mode) SetBool(v bool, m2 Mode) error {
	// must be the same mode (or off) when setting
	if *m != ModeOff && *m != m2 {
		return m.confliftErr(m2)
	}
	if !v {
		*m = ModeOff
	} else {
		*m = m2
	}
	return nil
}
func (m Mode) confliftErr(m2 Mode) error {
	return fmt.Errorf("conflicting emu mode: %v vs %v", m, m2)
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
