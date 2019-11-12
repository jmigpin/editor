package event

type KeyModifiers uint32

func (km KeyModifiers) HasAny(m KeyModifiers) bool {
	return km&m > 0
}
func (km KeyModifiers) Is(m KeyModifiers) bool {
	return km == m
}
func (km KeyModifiers) ClearLocks() KeyModifiers {
	w := []KeyModifiers{ModLock, ModNum}
	u := km
	for _, m := range w {
		u &^= m
	}
	return u
}

const (
	ModNone  KeyModifiers = 0
	ModShift KeyModifiers = 1 << (iota - 1)
	ModLock               // caps
	ModCtrl
	Mod1 // ~ alt
	Mod2 // ~ num lock
	Mod3
	Mod4 // ~ windows key
	Mod5 // ~ alt gr
)

const (
	ModAlt   = Mod1
	ModNum   = Mod2
	ModAltGr = Mod5
)

//----------

type KeySym int

const (
	KSymNone KeySym = 0

	// let ascii codes keep their values
	KSym_dummy_ KeySym = 256 + iota

	KSymSpace

	KSymBackspace
	KSymReturn
	KSymEscape
	KSymHome
	KSymLeft
	KSymUp
	KSymRight
	KSymDown
	KSymPageUp
	KSymPageDown
	KSymEnd
	KSymInsert
	KSymF1
	KSymF2
	KSymF3
	KSymF4
	KSymF5
	KSymF6
	KSymF7
	KSymF8
	KSymF9
	KSymF10
	KSymF11
	KSymF12

	KSymShiftL
	KSymShiftR
	KSymControlL
	KSymControlR
	KSymAltL
	KSymAltR
	KSymAltGr
	KSymSuperL // windows key
	KSymSuperR
	KSymDelete
	KSymTab
	KSymTabLeft

	KSymNumLock
	KSymCapsLock
	KSymShiftLock

	KSymGrave
	KSymAcute
	KSymCircumflex
	KSymTilde
	KSymMacron
	KSymBreve
	KSymDiaresis
	KSymRingAbove
	KSymCaron
	KSymCedilla

	KSymKeypadMultiply
	KSymKeypadAdd
	KSymKeypadSubtract
	KSymKeypadDecimal
	KSymKeypadDivide

	KSymKeypad0
	KSymKeypad1
	KSymKeypad2
	KSymKeypad3
	KSymKeypad4
	KSymKeypad5
	KSymKeypad6
	KSymKeypad7
	KSymKeypad8
	KSymKeypad9

	KSymVolumeUp
	KSymVolumeDown
	KSymMute

	KSymMultiKey
	KSymMenu
)
