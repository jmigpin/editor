//go:build windows

package windriver

import (
	"github.com/jmigpin/editor/util/uiutil/event"
)

func translateVKeyToEventKeySym(vkey uint32, ru rune) event.KeySym {
	ks := runeToEventKeySym(ru)
	if ks == 0 {
		ks = translateVKeyToEventKeySym2(vkey)
	}
	return ks
}

func translateVKeyToEventKeySym2(vkey uint32) event.KeySym {
	switch vkey {
	case 0x30:
		return event.KSym0
	case 0x31:
		return event.KSym1
	case 0x32:
		return event.KSym2
	case 0x33:
		return event.KSym3
	case 0x34:
		return event.KSym4
	case 0x35:
		return event.KSym5
	case 0x36:
		return event.KSym6
	case 0x37:
		return event.KSym7
	case 0x38:
		return event.KSym8
	case 0x39:
		return event.KSym9

	case 0x41:
		return event.KSymA
	case 0x42:
		return event.KSymB
	case 0x43:
		return event.KSymC
	case 0x44:
		return event.KSymD
	case 0x45:
		return event.KSymE
	case 0x46:
		return event.KSymF
	case 0x47:
		return event.KSymG
	case 0x48:
		return event.KSymH
	case 0x49:
		return event.KSymI
	case 0x4A:
		return event.KSymJ
	case 0x4B:
		return event.KSymK
	case 0x4C:
		return event.KSymL
	case 0x4D:
		return event.KSymM
	case 0x4E:
		return event.KSymN
	case 0x4F:
		return event.KSymO
	case 0x50:
		return event.KSymP
	case 0x51:
		return event.KSymQ
	case 0x52:
		return event.KSymR
	case 0x53:
		return event.KSymS
	case 0x54:
		return event.KSymT
	case 0x55:
		return event.KSymU
	case 0x56:
		return event.KSymV
	case 0x57:
		return event.KSymW
	case 0x58:
		return event.KSymX
	case 0x59:
		return event.KSymY
	case 0x5A:
		return event.KSymZ

	case 0x20:
		return event.KSymSpace
	case 0x08: // VK_BACK
		return event.KSymBackspace
	case 0x0D, 228: // TEST
		return event.KSymReturn
	case 0x1B:
		return event.KSymEscape
	case 0x24:
		return event.KSymHome
	case 0x25:
		return event.KSymLeft
	case 0x26:
		return event.KSymUp
	case 0x27:
		return event.KSymRight
	case 0x28:
		return event.KSymDown
	case 0x21: // VK_PRIOR
		return event.KSymPageUp
	case 0x22: // VK_NEXT
		return event.KSymPageDown
	case 0x23:
		return event.KSymEnd
	case 0x2D:
		return event.KSymInsert
	case 0xA0:
		return event.KSymShiftL
	case 0xA1:
		return event.KSymShiftR
	case 0xA2:
		return event.KSymControlL
	case 0xA3:
		return event.KSymControlR
	case 0xa4: // VK_LMENU
		return event.KSymAltL
	case 0xa5: // VK_RMENU
		return event.KSymAltR
	case 0x12: // VK_MENU
		return event.KSymAltGr // 227? 50?
	case 0x5B: // VK_LWIN: windows key
		return event.KSymSuperL
	case 0x5C: // VK_RWIN: windows key
		return event.KSymSuperR
	case 0x2E:
		return event.KSymDelete
	case 0x09: // VK_TAB
		return event.KSymTab
	//case 0x: return event.KSymTabLeft

	case 144: // 0x90:
		return event.KSymNumLock
	case 0x14: // VK_CAPITAL
		return event.KSymCapsLock
	//case 0x: return event.KSymShiftLock

	//case 0x21:
	//	return event.KSymExclam
	//case 0x22:
	//	return event.KSymDoubleQuote
	//case 0x23:
	//	return event.KSymNumberSign
	//case 0x24:
	//	return event.KSymDollar
	//case 0x25:
	//	return event.KSymPercent
	//case 0x26:
	//	return event.KSymAmpersand
	case 189: // TEST
		return event.KSymApostrophe
	//case 0x28:
	//	return event.KSymParentL
	//case 0x29:
	//	return event.KSymParentR
	case 0x6a: // VK_MULTIPLY
		return event.KSymAsterisk
	case 107, 219: // TEST
		return event.KSymPlus
	case 0xbc: // VK_OEM_COMMA
		return event.KSymComma
	case 109, 191: // TEST // 0xbd=VK_OEM_MINUS
		return event.KSymMinus
	case 110, 108, 190: // TEST // 0xbe=VK_OEM_PERIOD
		return event.KSymPeriod
	case 111: // TEST
		return event.KSymSlash
	case 192: // TEST
		return event.KSymBackSlash
	//case 0x3a:
	//	return event.KSymColon
	//case 0x3b:
	//	return event.KSymSemicolon
	case 226: // TEST
		return event.KSymLess
	case 146: // TEST
		return event.KSymEqual
	//case 0x3e:
	//	return event.KSymGreater
	//case 0xbf: // VK_OEM_2
	//	return event.KSymQuestion
	//case 0x40:
	//	return event.KSymAt
	//case 0x5b:
	//	return event.KSymBracketL
	//case 0x5d:
	//	return event.KSymBracketR

	//case 0xdd:
	//return event.KSymGrave
	case 221: // TEST
		return event.KSymAcute
	//case 0x: return event.KSymCircumflex
	case 220: // TEST
		return event.KSymTilde
	//case 0x: return event.KSymCedilla
	//case 0x: return event.KSymBreve
	//case 0x: return event.KSymCaron
	//case 0x: return event.KSymDiaresis
	//case 0x: return event.KSymRingAbove
	//case 0x: return event.KSymMacron

	case 0x70:
		return event.KSymF1
	case 0x71:
		return event.KSymF2
	case 0x72:
		return event.KSymF3
	case 0x73:
		return event.KSymF4
	case 0x74:
		return event.KSymF5
	case 0x75:
		return event.KSymF6
	case 0x76:
		return event.KSymF7
	case 0x77:
		return event.KSymF8
	case 0x78:
		return event.KSymF9
	case 0x79:
		return event.KSymF10
	case 0x7A:
		return event.KSymF11
	case 0x7B:
		return event.KSymF12

	case 0x60: // VK_NUMPAD0
		return event.KSymKeypad0
	case 0x61:
		return event.KSymKeypad1
	case 0x62:
		return event.KSymKeypad2
	case 0x63:
		return event.KSymKeypad3
	case 0x64:
		return event.KSymKeypad4
	case 0x65:
		return event.KSymKeypad5
	case 0x66:
		return event.KSymKeypad6
	case 0x67:
		return event.KSymKeypad7
	case 0x68:
		return event.KSymKeypad8
	case 0x69:
		return event.KSymKeypad9
	//case 0x6A: // VK_MULTIPLY: TODO: keypad?
	//return event.KSymKeypadMultiply
	//case 0x6B:
	//	return event.KSymKeypadAdd
	//case 0x6D:
	//	return event.KSymKeypadSubtract
	//case 0x6E:
	//	return event.KSymKeypadDecimal
	//case 0x6F:
	//	return event.KSymKeypadDivide

	case 0xAF:
		return event.KSymVolumeUp
	case 0xAE:
		return event.KSymVolumeDown
	case 0xAD:
		return event.KSymMute

		//case 0x: return event.KSymMultiKey
		//case 0x: return event.KSymMenu
	}
	return event.KSymNone
}

func runeToEventKeySym(ru rune) event.KeySym {
	switch ru {
	case '`':
		return event.KSymGrave
	case '´':
		return event.KSymAcute
	case '^':
		return event.KSymCircumflex
	case '~':
		return event.KSymTilde
	case '¸':
		return event.KSymCedilla
	case '˘':
		return event.KSymBreve
	case 'ˇ':
		return event.KSymCaron
	case '¨':
		return event.KSymDiaresis
	case '˚':
		return event.KSymRingAbove
	case '¯':
		return event.KSymMacron
	}
	return 0
}

//func eventKeySymToRune(ks event.KeySym) rune {
//	switch ks {
//	case event.KSymGrave:
//		return '`'
//		//case '´':
//		//	return event.KSymAcute
//		//case '^':
//		//	return event.KSymCircumflex
//		//case '~':
//		//	return event.KSymTilde
//		//case '¸':
//		//	return event.KSymCedilla
//		//case '˘':
//		//	return event.KSymBreve
//		//case 'ˇ':
//		//	return event.KSymCaron
//		//case '¨':
//		//	return event.KSymDiaresis
//		//case '˚':
//		//	return event.KSymRingAbove
//		//case '¯':
//		//	return event.KSymMacron
//	}
//	return 0
//}

//----------

const (
	kstateToggleBit = 1
	kstateDownBit   = 1 << (8 - 1)
)

//----------

func translateKStateToEventKeyModifiers(kstate *[256]byte) event.KeyModifiers {
	type pair struct {
		a byte
		b event.KeyModifiers
	}
	pairs := []pair{
		{_VK_SHIFT, event.ModShift},
		{_VK_CONTROL, event.ModCtrl},
		{_VK_MENU, event.ModAlt},
	}
	var w event.KeyModifiers
	for _, p := range pairs {
		if kstate[p.a]&kstateDownBit != 0 {
			w |= p.b
		}
	}
	return w
}

func translateKStateToEventMouseButtons(kstate *[256]byte) event.MouseButtons {
	type pair struct {
		a byte
		b event.MouseButton
	}
	pairs := []pair{
		{_VK_LBUTTON, event.ButtonLeft},
		{_VK_MBUTTON, event.ButtonMiddle},
		{_VK_RBUTTON, event.ButtonRight},
		{_VK_XBUTTON1, event.ButtonBackward},
		{_VK_XBUTTON2, event.ButtonForward},
	}
	var w event.MouseButtons
	for _, p := range pairs {
		if kstate[p.a]&kstateDownBit != 0 {
			w |= event.MouseButtons(p.b)
		}
	}
	return w
}

//----------

func translateVKeyToEventKeyModifiers(vkey uint32) event.KeyModifiers {
	type pair struct {
		a uint32
		b event.KeyModifiers
	}
	pairs := []pair{
		{_MK_SHIFT, event.ModShift},
		{_MK_CONTROL, event.ModCtrl},
	}
	var w event.KeyModifiers
	for _, p := range pairs {
		if vkey&p.a > 0 {
			w |= p.b
		}
	}
	return w
}

func translateVKeyToEventMouseButtons(vkey uint32) event.MouseButtons {
	type pair struct {
		a uint32
		b event.MouseButton
	}
	pairs := []pair{
		{_MK_LBUTTON, event.ButtonLeft},
		{_MK_MBUTTON, event.ButtonMiddle},
		{_MK_RBUTTON, event.ButtonRight},
		{_MK_XBUTTON1, event.ButtonBackward},
		{_MK_XBUTTON2, event.ButtonForward},
	}
	var w event.MouseButtons
	for _, p := range pairs {
		if vkey&p.a > 0 {
			w |= event.MouseButtons(p.b)
		}
	}
	return w
}

//----------

//type KeyData struct {
//	Count             int
//	ScanCode          int
//	Extended          bool
//	ContextCode       bool // always zero for keydown
//	PreviousStateDown bool
//	TransitionState   bool // always zero for keydown
//}

//func keyData(v uint32) *KeyData {
//	kd := &KeyData{}
//	kd.Count = int(v & 0xffff)              // bits: 0-15
//	kd.ScanCode = int((v & 0xff0000) >> 16) // bits: 16-23
//	kd.Extended = int(v&(1<<24)) != 0       // bits: 24
//	// bits 25-28 are reserved
//	kd.ContextCode = int(v&(1<<29)) != 0
//	kd.PreviousStateDown = int(v&(1<<30)) != 0
//	kd.TransitionState = int(v&(1<<31)) != 0
//	return kd
//}
