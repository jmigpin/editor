package event

//go:generate stringer -type KeySym -output zkeys.go

type KeySym int

const (
	KSymNone KeySym = 0

	// let ascii codes keep their values (adding 256 ensures gap)
	KSym_dummy_ KeySym = 256 + iota

	KSym0
	KSym1
	KSym2
	KSym3
	KSym4
	KSym5
	KSym6
	KSym7
	KSym8
	KSym9

	KSymA
	KSymB
	KSymC
	KSymD
	KSymE
	KSymF
	KSymG
	KSymH
	KSymI
	KSymJ
	KSymK
	KSymL
	KSymM
	KSymN
	KSymO
	KSymP
	KSymQ
	KSymR
	KSymS
	KSymT
	KSymU
	KSymV
	KSymW
	KSymX
	KSymY
	KSymZ

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
	KSymCapsLock  // only capitalizes letters
	KSymShiftLock // prints all keys secondary symbols

	KSymExclam      // !
	KSymDoubleQuote // "
	KSymNumberSign  // #
	KSymDollar      // $
	KSymPercent     // %
	KSymAmpersand   // &
	KSymApostrophe  // '
	KSymParentL     // (
	KSymParentR     // )
	KSymAsterisk    // *
	KSymPlus        // +
	KSymComma       // ,
	KSymMinus       // -
	KSymPeriod      // .
	KSymSlash       // /
	KSymBackSlash   // "\"
	KSymColon       // :
	KSymSemicolon   // ;
	KSymLess        // <
	KSymEqual       // =
	KSymGreater     // >
	KSymQuestion    // ?
	KSymAt          // @
	KSymBracketL    // [
	KSymBracketR    // ]

	KSymGrave      // `
	KSymAcute      // ´
	KSymCircumflex // ^
	KSymTilde      // ~
	KSymCedilla    // ¸
	KSymBreve      // ˘
	KSymCaron      // ˇ
	KSymDiaresis   // ¨
	KSymRingAbove  // ˚
	KSymMacron     // ¯

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
	KSymF13
	KSymF14
	KSymF15
	KSymF16

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
	KSymKeypadMultiply
	KSymKeypadAdd
	KSymKeypadSubtract
	KSymKeypadDecimal
	KSymKeypadDivide
	KSymKeypadEnter
	KSymKeypadSeparator
	KSymKeypadDelete

	KSymVolumeUp
	KSymVolumeDown
	KSymMute

	KSymMultiKey
	KSymMenu
)

//----------

type KeyModifiers uint16

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
	// TODO: rename to KMod
	ModNone  KeyModifiers = 0
	ModShift KeyModifiers = 1 << (iota - 1)
	ModLock               // caps // TODO: rename ModCapsLock
	ModCtrl
	Mod1 // ~ alt
	Mod2 // ~ num lock
	Mod3
	Mod4 // ~ windows key
	Mod5 // ~ alt gr
)
const (
	ModNum   = Mod2 // TODO: rename ModNumLock
	ModAlt   = Mod1
	ModAltGr = Mod5
)

//----------

type MouseButton uint16

const (
	ButtonNone MouseButton = 0
	ButtonLeft MouseButton = 1 << (iota - 1)
	ButtonMiddle
	ButtonRight
	ButtonWheelUp
	ButtonWheelDown
	ButtonWheelLeft
	ButtonWheelRight
	ButtonBackward // TODO: rename X1?
	ButtonForward  // TODO: rename X2?
)

//----------

type MouseButtons uint16

func (mb MouseButtons) Has(b MouseButton) bool {
	return mb&MouseButtons(b) > 0
}
func (mb MouseButtons) HasAny(bs MouseButtons) bool {
	return mb&bs > 0
}
func (mb MouseButtons) Is(b MouseButton) bool {
	return mb == MouseButtons(b)
}
