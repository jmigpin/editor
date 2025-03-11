package xinput

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// $ man keymaps
// https://tronche.com/gui/x/xlib/input/XGetKeyboardMapping.html
// https://tronche.com/gui/x/xlib/input/keyboard-encoding.html
// http://wiki.linuxquestions.org/wiki/List_of_Keysyms_Recognised_by_Xmodmap
// https://www.x.org/releases/X11R7.7/doc/libX11/i18n/compose/iso8859-2.html

// xproto.Keycode is a physical key.
// xproto.Keysym is the encoding of a symbol on the cap of a key.
// A list of keysyms is associated with each keycode.

//----------

// Keyboard mapping
type KMap struct {
	si    *xproto.SetupInfo
	reply *xproto.GetKeyboardMappingReply
	conn  *xgb.Conn

	modGroups struct {
		numLock byte
		altGr   byte
	}
}

func NewKMap(conn *xgb.Conn) (*KMap, error) {
	km := &KMap{conn: conn}
	err := km.ReadMapping()
	if err != nil {
		return nil, err
	}
	return km, nil
}

//----------

func (km *KMap) ReadMapping() error {
	if err := km.readKeyboardMapping(); err != nil {
		return err
	}
	if err := km.readModMapping(); err != nil {
		return err
	}
	return nil
}

func (km *KMap) readKeyboardMapping() error {
	si := xproto.Setup(km.conn)
	count := byte(si.MaxKeycode - si.MinKeycode + 1)
	if count <= 0 {
		return fmt.Errorf("bad keycode count: %v", count)
	}
	reply, err := xproto.GetKeyboardMapping(km.conn, si.MinKeycode, count).Reply()
	if err != nil {
		return err
	}
	if reply.KeysymsPerKeycode < 2 {
		return fmt.Errorf("keysyms per keycode < 2")
	}
	km.reply = reply
	km.si = si

	//log.Printf("%v", km.KeysymTable())

	return nil
}

func (km *KMap) readModMapping() error {
	modMap, err := xproto.GetModifierMapping(km.conn).Reply()
	if err != nil {
		return err
	}

	// 8 modifiers groups, that can have n keycodes
	//0	Shift (ModMaskShift)
	//1	Lock (Caps Lock)
	//2	Control (ModMaskControl)
	//3	Mod1 (Usually Alt)
	//4	Mod2 (Often Num Lock)
	//5	Mod3 (Rarely used)
	//6	Mod4 (Often Super/Meta)
	//7	Mod5 (Often AltGr)

	// X11
	const numLock = 0xff7f
	const altGr = 0xfe03 // TODO: alternatives 0xfe11, 0xff7e?

	// defaults
	km.modGroups.numLock = 4
	km.modGroups.altGr = 7

	// detect
	stride := modMap.KeycodesPerModifier
	for g := byte(0); g < 8; g++ {
		kcs := modMap.Keycodes[g*stride : (g+1)*stride]
		//fmt.Println(g, kcs) // DEBUG
	kcLoop: // iterate keycodes/keysyms, keep first found group
		for _, kc := range kcs { //
			kss := km.keycodeToKeysyms(kc)
			//fmt.Println("\t", kss) // DEBUG
			// iterate all modifiers, keep first
			for _, ks := range kss {
				switch ks {
				case numLock:
					km.modGroups.numLock = g
					break kcLoop
				case altGr:
					km.modGroups.altGr = g
					break kcLoop
				}
			}
		}
	}

	return nil
}

//----------

func (km *KMap) KeysymTable() string {
	o := "keysym table\n"
	for j := 0; j < 256; j++ {
		kc := xproto.Keycode(j)
		kss := km.keycodeToKeysyms(kc)
		u := []string{}
		for _, xks := range kss {
			eks := keysymToEventKeysym(xks)
			ru := eventKeysymRune(eks)
			u = append(u, fmt.Sprintf("\t(%c,%v)", ru, eks))
		}
		us := strings.Join(u, "\n")
		if len(us) > 0 {
			us = "\n" + us
		}
		o += fmt.Sprintf("kc=%v:%v\n", kc, us)
	}
	return o
}

//----------

func (km *KMap) keycodeToKeysyms(keycode xproto.Keycode) []xproto.Keysym {
	y := int(keycode - km.si.MinKeycode)
	n := km.si.MaxKeycode - km.si.MinKeycode + 1
	if y < 0 || y >= int(n) {
		return nil
	}
	stride := int(km.reply.KeysymsPerKeycode) // usually ~7
	return km.reply.Keysyms[y*stride : (y+1)*stride]
}

//----------

func (km *KMap) printKeysyms(keycode xproto.Keycode) {
	keysyms := km.keycodeToKeysyms(keycode)
	//fmt.Printf("%v\n", keysyms)

	{
		u := []string{}
		for _, ks := range keysyms {
			u = append(u, string(rune(ks)))
		}
		fmt.Printf("[%v]\n", strings.Join(u, " "))
	}
	{
		u := []string{}
		for _, ks := range keysyms {
			u = append(u, fmt.Sprintf("%x", ks))
		}
		fmt.Printf("[%v]\n", strings.Join(u, " "))
	}
}

//----------

func (km *KMap) keysymsToKeysym(kss []xproto.Keysym, m uint16) xproto.Keysym {
	bitIsSet := func(v uint16) bool { return m&v > 0 }
	hasShift := bitIsSet(xproto.KeyButMaskShift)
	hasCaps := bitIsSet(xproto.KeyButMaskLock)
	hasCtrl := bitIsSet(xproto.KeyButMaskControl)
	//hasNum := bitIsSet(xproto.KeyButMaskMod2)
	hasNum := bitIsSet(1 << km.modGroups.numLock) // detected
	//hasAltGr := bitIsSet(xproto.KeyButMaskMod5)
	hasAltGr := bitIsSet(1 << km.modGroups.altGr) // detected

	// keysym group
	group := 0
	if hasCtrl {
		group = 1
	} else if hasAltGr {
		group = 2
	}

	// each group has two symbols
	i1 := group * 2
	i2 := i1 + 1
	if i1 >= len(kss) {
		return 0
	}
	if i2 >= len(kss) {
		i2 = i1
	}
	ks1, ks2 := kss[i1], kss[i2]
	if ks2 == 0 {
		ks2 = ks1
	}

	// keypad
	if hasNum && isKeypad(ks2) {
		if hasShift {
			return ks1
		} else {
			return ks2
		}
	}

	r1 := rune(ks1)
	hasLower := unicode.IsLower(unicode.ToLower(r1))

	if hasLower {
		shifted := (hasShift && !hasCaps) || (!hasShift && hasCaps)
		if shifted {
			return ks2
		}
		return ks1
	}

	if hasShift {
		return ks2
	}
	return ks1
}

//----------

func (km *KMap) Lookup(keycode xproto.Keycode, kmods uint16) (event.KeySym, rune) {
	kss := km.keycodeToKeysyms(keycode)
	ks := km.keysymsToKeysym(kss, kmods)
	eks := keysymToEventKeysym(ks)
	ru := keysymRune(ks, eks)
	return eks, ru
}

//----------
//----------
//----------

func isKeypad(ks xproto.Keysym) bool {
	return (0xFF80 <= ks && ks <= 0xFFBD) ||
		(0x11000000 <= ks && ks <= 0x1100FFFF)
}
