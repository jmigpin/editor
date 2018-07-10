package xinput

import (
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

var runeKeyCodeMap = map[xproto.Keysym]event.KeySym{
	xkISOLevel3Shift: event.KSymAltGr,
	xkISOLeftTab:     event.KSymTabLeft,

	xkBackSpace: event.KSymBackspace,
	xkReturn:    event.KSymReturn,
	xkEscape:    event.KSymEscape,
	xkHome:      event.KSymHome,
	xkLeft:      event.KSymLeft,
	xkUp:        event.KSymUp,
	xkRight:     event.KSymRight,
	xkDown:      event.KSymDown,
	xkPageUp:    event.KSymPageUp,
	xkPageDown:  event.KSymPageDown,
	xkEnd:       event.KSymEnd,
	xkInsert:    event.KSymInsert,
	xkF1:        event.KSymF1,
	xkF2:        event.KSymF2,
	xkF3:        event.KSymF3,
	xkF4:        event.KSymF4,
	xkF5:        event.KSymF5,
	xkF6:        event.KSymF6,
	xkF7:        event.KSymF7,
	xkF8:        event.KSymF8,
	xkF9:        event.KSymF9,
	xkF10:       event.KSymF10,
	xkF11:       event.KSymF11,
	xkF12:       event.KSymF12,
	xkShiftL:    event.KSymShiftL,
	xkShiftR:    event.KSymShiftR,
	xkControlL:  event.KSymControlL,
	xkControlR:  event.KSymControlR,
	xkAltL:      event.KSymAltL,
	xkAltR:      event.KSymAltR,
	xkSuperL:    event.KSymSuperL, // windows key
	xkSuperR:    event.KSymSuperR,
	xkDelete:    event.KSymDelete,
	xkTab:       event.KSymTab,

	xkNumLock:   event.KSymNumLock,
	xkCapsLock:  event.KSymCapsLock,
	xkShiftLock: event.KSymShiftLock,

	xkKeypadMultiply: event.KSymKeypadMultiply,
	xkKeypadAdd:      event.KSymKeypadAdd,
	xkKeypadSubtract: event.KSymKeypadSubtract,
	xkKeypadDecimal:  event.KSymKeypadDecimal,
	xkKeypadDivide:   event.KSymKeypadDivide,

	xkKeypad0: event.KSymKeypad0,
	xkKeypad1: event.KSymKeypad1,
	xkKeypad2: event.KSymKeypad2,
	xkKeypad3: event.KSymKeypad3,
	xkKeypad4: event.KSymKeypad4,
	xkKeypad5: event.KSymKeypad5,
	xkKeypad6: event.KSymKeypad6,
	xkKeypad7: event.KSymKeypad7,
	xkKeypad8: event.KSymKeypad8,
	xkKeypad9: event.KSymKeypad9,

	xf86xkAudioLowerVolume: event.KSymVolumeDown,
	xf86xkAudioRaiseVolume: event.KSymVolumeUp,
	xf86xkAudioMute:        event.KSymMute,

	//xkISOLeftTab     : event.KSym,
	//xkMultiKey  : event.KSym,
	//xkMetaLeft:  event.KSym,
	//xkMetaRight: event.KSym,
	//xkMenu      : event.KSym,
}

//----------

// Constants from /usr/include/X11/keysymdef.h
const (
	xkVoidSymbol = 0xffffff

	xkModeSwitch = 0xff7e

	xkISOLevel3Shift = 0xfe03 // alt gr?
	xkISOLeftTab     = 0xfe20

	xkBackSpace = 0xff08
	xkReturn    = 0xff0d
	xkEscape    = 0xff1b
	xkMultiKey  = 0xff20
	xkHome      = 0xff50
	xkLeft      = 0xff51
	xkUp        = 0xff52
	xkRight     = 0xff53
	xkDown      = 0xff54
	xkPageUp    = 0xff55
	xkPageDown  = 0xff56
	xkEnd       = 0xff57
	xkInsert    = 0xff63
	xkMenu      = 0xff67
	xkNumLock   = 0xff7f
	xkF1        = 0xffbe
	xkF2        = 0xffbf
	xkF3        = 0xffc0
	xkF4        = 0xffc1
	xkF5        = 0xffc2
	xkF6        = 0xffc3
	xkF7        = 0xffc4
	xkF8        = 0xffc5
	xkF9        = 0xffc6
	xkF10       = 0xffc7
	xkF11       = 0xffc8
	xkF12       = 0xffc9
	xkShiftL    = 0xffe1
	xkShiftR    = 0xffe2
	xkControlL  = 0xffe3
	xkControlR  = 0xffe4
	xkCapsLock  = 0xffe5
	xkShiftLock = 0xffe6
	xkMetaLeft  = 0xffe7
	xkMetaRight = 0xffe8
	xkAltL      = 0xffe9
	xkAltR      = 0xffea
	xkSuperL    = 0xffeb // windows key
	xkSuperR    = 0xffec
	xkDelete    = 0xffff
	xkTab       = 0xff09

	xkGrave      = 0xfe50
	xkAcute      = 0xfe51
	xkCircumflex = 0xfe52
	xkTilde      = 0xfe53

	xf86xkAudioLowerVolume = 0x1008ff11
	xf86xkAudioMute        = 0x1008ff12
	xf86xkAudioRaiseVolume = 0x1008ff13

	xkKeypadSpace     = 0xff80
	xkKeypadTab       = 0xff89
	xkKeypadEnter     = 0xff8d
	xkKeypadF1        = 0xff91
	xkKeypadF2        = 0xff92
	xkKeypadF3        = 0xff93
	xkKeypadF4        = 0xff94
	xkKeypadHome      = 0xff95
	xkKeypadLeft      = 0xff96
	xkKeypadUp        = 0xff97
	xkKeypadRight     = 0xff98
	xkKeypadDown      = 0xff99
	xkKeypadPrior     = 0xff9a
	xkKeypadPage_Up   = 0xff9a
	xkKeypadNext      = 0xff9b
	xkKeypadPage_Down = 0xff9b
	xkKeypadEnd       = 0xff9c
	xkKeypadBegin     = 0xff9d
	xkKeypadInsert    = 0xff9e
	xkKeypadDelete    = 0xff9f
	xkKeypadEqual     = 0xffbd
	xkKeypadMultiply  = 0xffaa
	xkKeypadAdd       = 0xffab
	xkKeypadSeparator = 0xffac // Separator, often comma
	xkKeypadSubtract  = 0xffad
	xkKeypadDecimal   = 0xffae
	xkKeypadDivide    = 0xffaf

	xkKeypad0 = 0xffb0
	xkKeypad1 = 0xffb1
	xkKeypad2 = 0xffb2
	xkKeypad3 = 0xffb3
	xkKeypad4 = 0xffb4
	xkKeypad5 = 0xffb5
	xkKeypad6 = 0xffb6
	xkKeypad7 = 0xffb7
	xkKeypad8 = 0xffb8
	xkKeypad9 = 0xffb9
)
