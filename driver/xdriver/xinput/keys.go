package xinput

import (
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// Constants from /usr/include/X11/keysymdef.h
func xkeysymToEventKeySym(xk int) event.KeySym {
	switch xk {
	case 0x0030:
		return event.KSym0
	case 0x0031:
		return event.KSym1
	case 0x0032:
		return event.KSym2
	case 0x0033:
		return event.KSym3
	case 0x0034:
		return event.KSym4
	case 0x0035:
		return event.KSym5
	case 0x0036:
		return event.KSym6
	case 0x0037:
		return event.KSym7
	case 0x0038:
		return event.KSym8
	case 0x0039:
		return event.KSym9

	case 0x0041:
		return event.KSymA
	case 0x0042:
		return event.KSymB
	case 0x0043:
		return event.KSymC
	case 0x0044:
		return event.KSymD
	case 0x0045:
		return event.KSymE
	case 0x0046:
		return event.KSymF
	case 0x0047:
		return event.KSymG
	case 0x0048:
		return event.KSymH
	case 0x0049:
		return event.KSymI
	case 0x004a:
		return event.KSymJ
	case 0x004b:
		return event.KSymK
	case 0x004c:
		return event.KSymL
	case 0x004d:
		return event.KSymM
	case 0x004e:
		return event.KSymN
	case 0x004f:
		return event.KSymO
	case 0x0050:
		return event.KSymP
	case 0x0051:
		return event.KSymQ
	case 0x0052:
		return event.KSymR
	case 0x0053:
		return event.KSymS
	case 0x0054:
		return event.KSymT
	case 0x0055:
		return event.KSymU
	case 0x0056:
		return event.KSymV
	case 0x0057:
		return event.KSymW
	case 0x0058:
		return event.KSymX
	case 0x0059:
		return event.KSymY
	case 0x005a:
		return event.KSymZ

	case 0x0020:
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
