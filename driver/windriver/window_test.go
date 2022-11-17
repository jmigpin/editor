//go:build windows

package windriver

import (
	"testing"

	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestVirtualKeyLookup(t *testing.T) {
	type pair struct {
		vkey   uint32
		kstate *[256]byte
		eks    event.KeySym
		ru     rune
	}

	kstate0 := [256]byte{}
	kstateShift := [256]byte{_VK_SHIFT: kstateDownBit}
	kstateCtrl := [256]byte{_VK_CONTROL: kstateDownBit}
	kstateCaps := [256]byte{_VK_CAPITAL: kstateToggleBit}
	kstateAlt := [256]byte{_VK_MENU: kstateDownBit}
	//kstateAltGr := [256]byte{_VK_SHIFT: kstateDownBit, _VK_CONTROL: kstateDownBit}
	//kstateAltGr := [256]byte{_VK_RMENU: kstateDownBit}
	//kstateAltGr := [256]byte{_VK_LMENU: kstateDownBit}
	_ = kstate0
	_ = kstateShift
	_ = kstateCtrl
	_ = kstateCaps
	_ = kstateAlt
	//_ = kstateAltGr

	//kstateTest1 := [256]byte{1: 1, 144: 129}
	//kstateTest1 := [256]byte{1: 1, 144: 1, 252: 129}

	pairs := []pair{
		{50, &kstate0, event.KSym2, '2'},
		{65, &kstate0, event.KSymA, 'a'},
		{65, &kstateShift, event.KSymA, 'A'},
		{65, &kstateCaps, event.KSymA, 'A'},
		{32, &kstate0, event.KSymSpace, ' '},
		{221, &kstate0, event.KSymAcute, '´'},
		{221, &kstateShift, event.KSymGrave, '`'},
		{220, &kstate0, event.KSymTilde, '~'},
		{220, &kstateShift, event.KSymCircumflex, '^'},

		// TODO
		//{220, &kstateTest1, 0, '^'},
		//{65, &kstateCtrl, event.KSymA, 'a'},
		//{0x32, &kstateAltGr, event.KSymAt, '@'}, // TODO
		//{35, &kstateShift, event.KSymGrave, '`'},
	}
	_ = pairs
	//for i := 0; i < 256; i++ {
	//p := pair{uint32(i), &kstate0, -1, -1}
	for _, p := range pairs {
		ru, _ := vkeyRune(p.vkey, p.kstate)
		eks := translateVKeyToEventKeySym(p.vkey, ru)
		if eks != p.eks || ru != p.ru {
			t.Logf("%v, %v, %v(%c)\n", p.vkey, eks, ru, ru)
			t.Fail()
		}
	}
}

//----------
