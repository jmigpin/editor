package event

type KeySym int

const (
	KSymNone KeySym = 0

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

	// let ascii codes keep their values (adding 256 ensures gap)
	KSym_dummy_ KeySym = 256 + iota

	KSymSpace
	KSymExclam      // !
	KSymDoubleQuote // "
	KSymNumberSign  // #
	KSymDollar      // $
	KSymPercent     // %
	KSymAmpersand   // &
	//KSymApostrophe  // ' // TODO
	//KSymQuoteRight  // ´ // TODO
	KSymParentL   // (
	KSymParentR   // )
	KSymAsterisk  // *
	KSymPlus      // +
	KSymComma     // ,
	KSymMinus     // -
	KSymPeriod    // .
	KSymSlash     // /
	KSymColon     // :
	KSymSemicolon // ;
	KSymLess      // <
	KSymEqual     // =
	KSymGreater   // >
	KSymQuestion  // ?
	KSymAt        // @
	kSymBracketL  // [
	kSymBracketR  // ]

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
	KSymCapsLock
	KSymShiftLock

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

	KSymVolumeUp
	KSymVolumeDown
	KSymMute

	KSymMultiKey
	KSymMenu
)

//----------

func KeySymRune(ks, ks2 KeySym, ru rune) (KeySym, rune) {
	if ks2 != KSymNone {
		ks = ks2
		ru2 := keySymRune2(ks2)
		if ru2 != 0 {
			ru = ru2
		}
	}
	return ks, ru
}

func keySymRune2(ks KeySym) rune {
	switch ks {
	case KSymGrave:
		return '`'
	case KSymAcute:
		return '´'
	case KSymCircumflex:
		return '^'
	case KSymTilde:
		return '~'
	case KSymCedilla:
		return '¸' // 0xb8
	case KSymBreve:
		return '˘' // 0x2d8
	case KSymCaron:
		return 'ˇ' // 0x2c7
	case KSymDiaresis:
		return '¨' // 0xa8
	case KSymRingAbove:
		return '˚' // 0x2da
	case KSymMacron:
		return '¯' // 0xaf

	case KSymKeypad0:
		return '0'
	case KSymKeypad1:
		return '1'
	case KSymKeypad2:
		return '2'
	case KSymKeypad3:
		return '3'
	case KSymKeypad4:
		return '4'
	case KSymKeypad5:
		return '5'
	case KSymKeypad6:
		return '6'
	case KSymKeypad7:
		return '7'
	case KSymKeypad8:
		return '8'
	case KSymKeypad9:
		return '9'

	case KSymKeypadMultiply:
		return '*'
	case KSymKeypadAdd:
		return '+'
	case KSymKeypadSubtract:
		return '-'
	case KSymKeypadDecimal:
		return '.'
	case KSymKeypadDivide:
		return '/'
	}
	return rune(0)
}

//----------

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
