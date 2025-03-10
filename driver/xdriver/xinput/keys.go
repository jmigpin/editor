package xinput

import (
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// Constants from /usr/include/X11/keysymdef.h
func translateXKeysymToEventKeySym(xk xproto.Keysym) event.KeySym {
	switch xk {
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

	case 0x41, 0x61:
		return event.KSymA
	case 0x42, 0x62:
		return event.KSymB
	case 0x43, 0x63:
		return event.KSymC
	case 0x44, 0x64:
		return event.KSymD
	case 0x45, 0x65:
		return event.KSymE
	case 0x46, 0x66:
		return event.KSymF
	case 0x47, 0x67:
		return event.KSymG
	case 0x48, 0x68:
		return event.KSymH
	case 0x49, 0x69:
		return event.KSymI
	case 0x4a, 0x6a:
		return event.KSymJ
	case 0x4b, 0x6b:
		return event.KSymK
	case 0x4c, 0x6c:
		return event.KSymL
	case 0x4d, 0x6d:
		return event.KSymM
	case 0x4e, 0x6e:
		return event.KSymN
	case 0x4f, 0x6f:
		return event.KSymO
	case 0x50, 0x70:
		return event.KSymP
	case 0x51, 0x71:
		return event.KSymQ
	case 0x52, 0x72:
		return event.KSymR
	case 0x53, 0x73:
		return event.KSymS
	case 0x54, 0x74:
		return event.KSymT
	case 0x55, 0x75:
		return event.KSymU
	case 0x56, 0x76:
		return event.KSymV
	case 0x57, 0x77:
		return event.KSymW
	case 0x58, 0x78:
		return event.KSymX
	case 0x59, 0x79:
		return event.KSymY
	case 0x5a, 0x7a:
		return event.KSymZ

	case 0x20:
		return event.KSymSpace
	case 0xff08:
		return event.KSymBackspace
	case 0xff0d:
		return event.KSymReturn
	case 0xff1b:
		return event.KSymEscape
	case 0xff50:
		return event.KSymHome
	case 0xff51:
		return event.KSymLeft
	case 0xff52:
		return event.KSymUp
	case 0xff53:
		return event.KSymRight
	case 0xff54:
		return event.KSymDown
	case 0xff55:
		return event.KSymPageUp
	case 0xff56:
		return event.KSymPageDown
	case 0xff57:
		return event.KSymEnd
	case 0xff63:
		return event.KSymInsert
	case 0xffe1:
		return event.KSymShiftL
	case 0xffe2:
		return event.KSymShiftR
	case 0xffe3:
		return event.KSymControlL
	case 0xffe4:
		return event.KSymControlR
	case 0xffe9:
		return event.KSymAltL
	case 0xffea:
		return event.KSymAltR
	case 0xfe03:
		return event.KSymAltGr // ISOLevel3Shift
	case 0xffeb:
		return event.KSymSuperL // windows key
	case 0xffec:
		return event.KSymSuperR
	case 0xffff:
		return event.KSymDelete
	case 0xff09:
		return event.KSymTab
	case 0xfe20:
		return event.KSymTabLeft // ISOLeftTab

	case 0xff7f:
		return event.KSymNumLock
	case 0xffe5:
		return event.KSymCapsLock
	case 0xffe6:
		return event.KSymShiftLock

	case 0x21:
		return event.KSymExclam
	case 0x22:
		return event.KSymDoubleQuote
	case 0x23:
		return event.KSymNumberSign
	case 0x24:
		return event.KSymDollar
	case 0x25:
		return event.KSymPercent
	case 0x26:
		return event.KSymAmpersand
	case 0x27:
		return event.KSymApostrophe
	case 0x28:
		return event.KSymParentL
	case 0x29:
		return event.KSymParentR
	case 0x2a:
		return event.KSymAsterisk
	case 0x2b:
		return event.KSymPlus
	case 0x2c:
		return event.KSymComma
	case 0x2d:
		return event.KSymMinus
	case 0x2e:
		return event.KSymPeriod
	case 0x2f:
		return event.KSymSlash
	case 0x5c:
		return event.KSymBackSlash
	case 0x3a:
		return event.KSymColon
	case 0x3b:
		return event.KSymSemicolon
	case 0x3c:
		return event.KSymLess
	case 0x3d:
		return event.KSymEqual
	case 0x3e:
		return event.KSymGreater
	case 0x3f:
		return event.KSymQuestion
	case 0x40:
		return event.KSymAt
	case 0x5b:
		return event.KSymBracketL
	case 0x5d:
		return event.KSymBracketR

	case 0xfe50:
		return event.KSymGrave
	case 0xfe51:
		return event.KSymAcute
	case 0xfe52:
		return event.KSymCircumflex
	case 0xfe53:
		return event.KSymTilde
	case 0xfe5b:
		return event.KSymCedilla
	case 0xfe55:
		return event.KSymBreve
	case 0xfe5a:
		return event.KSymCaron
	case 0xfe57:
		return event.KSymDiaresis
	case 0xfe58:
		return event.KSymRingAbove
	case 0xfe54:
		return event.KSymMacron

	case 0xffbe:
		return event.KSymF1
	case 0xffbf:
		return event.KSymF2
	case 0xffc0:
		return event.KSymF3
	case 0xffc1:
		return event.KSymF4
	case 0xffc2:
		return event.KSymF5
	case 0xffc3:
		return event.KSymF6
	case 0xffc4:
		return event.KSymF7
	case 0xffc5:
		return event.KSymF8
	case 0xffc6:
		return event.KSymF9
	case 0xffc7:
		return event.KSymF10
	case 0xffc8:
		return event.KSymF11
	case 0xffc9:
		return event.KSymF12

	case 0xffb0:
		return event.KSymKeypad0
	case 0xffb1:
		return event.KSymKeypad1
	case 0xffb2:
		return event.KSymKeypad2
	case 0xffb3:
		return event.KSymKeypad3
	case 0xffb4:
		return event.KSymKeypad4
	case 0xffb5:
		return event.KSymKeypad5
	case 0xffb6:
		return event.KSymKeypad6
	case 0xffb7:
		return event.KSymKeypad7
	case 0xffb8:
		return event.KSymKeypad8
	case 0xffb9:
		return event.KSymKeypad9
	case 0xffaa:
		return event.KSymKeypadMultiply
	case 0xffab:
		return event.KSymKeypadAdd
	case 0xffad:
		return event.KSymKeypadSubtract
	case 0xffae:
		return event.KSymKeypadDecimal
	case 0xffaf:
		return event.KSymKeypadDivide
	case 0xff8d:
		return event.KSymKeypadEnter
	case 0xffac:
		return event.KSymKeypadSeparator
	case 0xff9f:
		return event.KSymKeypadDelete

	case 0x1008ff13:
		return event.KSymVolumeUp
	case 0x1008ff11:
		return event.KSymVolumeDown
	case 0x1008ff12:
		return event.KSymMute

	case 0xff20:
		return event.KSymMultiKey
	case 0xff67:
		return event.KSymMenu
	}
	return event.KSymNone
}

//----------

func keySymsRune(xks xproto.Keysym, eks event.KeySym) rune {
	ru := rune(xks) // default direct translation (covers some ascii values)
	ru2 := eventKeySymRune(eks)
	if ru2 != 0 {
		ru = ru2
	}
	return ru
}

func eventKeySymRune(eks event.KeySym) rune {
	switch eks {
	case event.KSymGrave:
		return '`'
	case event.KSymAcute:
		return '´'
	case event.KSymCircumflex:
		return '^'
	case event.KSymTilde:
		return '~'
	case event.KSymCedilla:
		return '¸' // 0xb8
	case event.KSymBreve:
		return '˘' // 0x2d8
	case event.KSymCaron:
		return 'ˇ' // 0x2c7
	case event.KSymDiaresis:
		return '¨' // 0xa8
	case event.KSymRingAbove:
		return '˚' // 0x2da
	case event.KSymMacron:
		return '¯' // 0xaf

	case event.KSymKeypad0:
		return '0'
	case event.KSymKeypad1:
		return '1'
	case event.KSymKeypad2:
		return '2'
	case event.KSymKeypad3:
		return '3'
	case event.KSymKeypad4:
		return '4'
	case event.KSymKeypad5:
		return '5'
	case event.KSymKeypad6:
		return '6'
	case event.KSymKeypad7:
		return '7'
	case event.KSymKeypad8:
		return '8'
	case event.KSymKeypad9:
		return '9'

	case event.KSymKeypadMultiply:
		return '*'
	case event.KSymKeypadAdd:
		return '+'
	case event.KSymKeypadSubtract:
		return '-'
	case event.KSymKeypadDecimal:
		return '.'
	case event.KSymKeypadDivide:
		return '/'
	}
	return rune(0)
}

//----------

func translateModifiersToEventKeyModifiers(v uint16) event.KeyModifiers {
	type pair struct {
		a uint16
		b event.KeyModifiers
	}
	pairs := []pair{
		{xproto.KeyButMaskShift, event.ModShift},
		{xproto.KeyButMaskControl, event.ModCtrl},
		{xproto.KeyButMaskLock, event.ModLock},
		{xproto.KeyButMaskMod1, event.Mod1},
		{xproto.KeyButMaskMod2, event.Mod2},
		{xproto.KeyButMaskMod3, event.Mod3},
		{xproto.KeyButMaskMod4, event.Mod4},
		{xproto.KeyButMaskMod5, event.Mod5},
	}
	var w event.KeyModifiers
	for _, p := range pairs {
		if v&p.a > 0 {
			w |= p.b
		}
	}
	return w
}

func translateModifiersToEventMouseButtons(v uint16) event.MouseButtons {
	type pair struct {
		a uint16
		b event.MouseButton
	}
	pairs := []pair{
		{xproto.KeyButMaskButton1, event.ButtonLeft},
		{xproto.KeyButMaskButton2, event.ButtonMiddle},
		{xproto.KeyButMaskButton3, event.ButtonRight},
		{xproto.KeyButMaskButton4, event.ButtonWheelUp},
		{xproto.KeyButMaskButton5, event.ButtonWheelDown},
		{xproto.KeyButMaskButton5 << 1, event.ButtonWheelLeft},
		{xproto.KeyButMaskButton5 << 2, event.ButtonWheelRight},
		{xproto.KeyButMaskButton5 << 3, event.ButtonBackward},
		// event sends uint16 (no support for forward?)
		//{xproto.KeyButMaskButton5 << 4, event.ButtonForward},
	}
	var w event.MouseButtons
	for _, p := range pairs {
		if v&p.a > 0 {
			w |= event.MouseButtons(p.b)
		}
	}
	return w
}

func translateButtonToEventButton(xb xproto.Button) event.MouseButton {
	var b event.MouseButton
	switch xb {
	case 1:
		b = event.ButtonLeft
	case 2:
		b = event.ButtonMiddle
	case 3:
		b = event.ButtonRight
	case 4:
		b = event.ButtonWheelUp
	case 5:
		b = event.ButtonWheelDown
	case 6:
		b = event.ButtonWheelLeft
	case 7:
		b = event.ButtonWheelRight
	case 8:
		b = event.ButtonBackward
	case 9:
		b = event.ButtonForward
	}
	return b
}
