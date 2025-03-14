package xinput

import (
	"bytes"
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
// https://www.x.org/releases/X11R7.7/doc/kbproto/xkbproto.html#Transforming_the_KeySym_Associated_with_a_Key_Event

// xproto.Keycode is a physical key.
// xproto.Keysym is the encoding of a symbol on the cap of a key.
// A list of keysyms is associated with each keycode.

//----------

// Keyboard mapping
type KMap struct {
	conn  *xgb.Conn
	si    *xproto.SetupInfo
	kbm   [256][]xproto.Keysym // keyboard map
	mmask struct {             // modifiers masks
		shift     uint16
		capsL     uint16
		ctrl      uint16
		numL      uint16
		alt       uint16
		altGr     uint16
		superMeta uint16
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
	if err := km.readModifiersMapping(); err != nil {
		return err
	}
	return nil
}

func (km *KMap) readKeyboardMapping() error {
	si := xproto.Setup(km.conn)
	count := byte(si.MaxKeycode - si.MinKeycode + 1)
	reply, err := xproto.GetKeyboardMapping(km.conn, si.MinKeycode, count).Reply()
	if err != nil {
		return err
	}

	stride := int(reply.KeysymsPerKeycode)
	for i := 0; i < 256; i++ {
		if i >= int(si.MinKeycode) && i <= int(si.MaxKeycode) {
			k := i - int(si.MinKeycode)
			w := reply.Keysyms[k*stride : (k+1)*stride]
			km.kbm[i] = w
		}
	}

	//fmt.Println(km.kbm)

	return nil
}

func (km *KMap) setKeyboardMappingEntries(kbm map[xproto.Keycode][]xproto.Keysym) {
	for kc, kss := range kbm {
		km.kbm[kc] = kss
	}
}

func (km *KMap) readModifiersMapping() error {
	modMap, err := xproto.GetModifierMapping(km.conn).Reply()
	if err != nil {
		return err
	}

	mm := [8][]xproto.Keycode{}
	stride := int8(modMap.KeycodesPerModifier)
	for g := int8(0); g < 8; g++ {
		w := modMap.Keycodes[g*stride : (g+1)*stride]
		mm[g] = w
	}

	km.detectModifiersMapping(mm)
	return nil
}

func (km *KMap) detectModifiersMapping(mm [8][]xproto.Keycode) {
	// keysyms to detect which group might have them
	numLocks := []xproto.Keysym{
		0xff7f, // XK_Num_Lock
	}
	alts := []xproto.Keysym{
		0xffe9, // XK_Alt_L
		0xffea, // XK_Alt_R
	}
	altGrs := []xproto.Keysym{
		0xfe03, // XK_ISO_Level3_Shift
		0xfe11, // XK_ISO_Level5_Shift
		0xff7e, // XK_ISO_Group_Shift
	}
	superMetas := []xproto.Keysym{
		0xffeb, // XK_Super_L
		0xffe7, // XK_Meta_L
		0xffec, // XK_Super_R
		0xffe8, // XK_Meta_R
	}

	// defaults
	km.mmask.shift = xproto.KeyButMaskShift
	km.mmask.capsL = xproto.KeyButMaskLock
	km.mmask.ctrl = xproto.KeyButMaskControl
	km.mmask.alt = 1 << 3  // mod1
	km.mmask.numL = 1 << 4 // mod2
	// mod3 // rarely used
	km.mmask.superMeta = 0  // mod4
	km.mmask.altGr = 1 << 7 // mod 5

	type pair struct {
		group *uint16
		kss   []xproto.Keysym
	}

	pairs := []pair{
		pair{&km.mmask.numL, numLocks},
		pair{&km.mmask.alt, alts},
		pair{&km.mmask.altGr, altGrs},
		pair{&km.mmask.superMeta, superMetas},
	}

	// 8 modifiers groups, that can have n keycodes
	//0	Shift
	//1	Lock (Caps Lock)
	//2	Control
	//--- detect
	//3	Mod1 (Usually Alt)
	//4	Mod2 (Often Num Lock)
	//5	Mod3 (Rarely used)
	//6	Mod4 (Often Super/Meta)
	//7	Mod5 (Often AltGr)

	// detect
	for g, kcs := range mm {
	kcLoop: // iterate keycodes/keysyms, keep first found group
		for _, kc := range kcs { //
			kss := km.kbm[kc]
			//fmt.Println("\t", kss) // DEBUG
			for _, ks := range kss {
				for pi, p := range pairs {
					_ = pi
					for _, ks2 := range p.kss {
						if ks == ks2 {
							*p.group = 1 << g
							//fmt.Println("g", g, "detected", fmt.Sprintf("%x", ks), "pair", pi) // DEBUG
							break kcLoop
						}
					}
				}
			}
		}
	}
}

//----------

func (km *KMap) Lookup(kc xproto.Keycode, kmods uint16) (xproto.Keysym, event.KeySym, rune) {
	kss := km.kbm[kc]
	ks := km.keysymsToKeysym(kss, kmods)
	eks := keysymToEventKeysym(ks)
	ru := finalKeysymRune(eks, ks)
	return ks, eks, ru
}

//----------

func (km *KMap) keysymsToKeysym(kss []xproto.Keysym, m uint16) xproto.Keysym {
	em := km.modifiersToEventModifiers(m)

	hasShift := em.HasAny(event.ModShift)
	hasCtrl := em.HasAny(event.ModCtrl)
	hasAltGr := em.HasAny(event.ModAltGr)
	hasCapsLock := em.HasAny(event.ModCapsLock)
	hasNumLock := em.HasAny(event.ModNumLock)

	// keysym group
	group := 0
	if hasAltGr {
		group = 2
		if hasCtrl { // TODO: this is custom, since ctrl alone is probably not the correct group changer
			group = 1
		}
	}

	// each group has two symbols (normal and shifted)
	i1 := group * 2
	i2 := i1 + 1
	if i1 >= len(kss) {
		return 0
	}
	i2v := xproto.Keysym(0)
	if i2 < len(kss) {
		i2v = kss[i2]
	}
	ks1, ks2 := kss[i1], i2v

	// canZero means it can return zero (no action), honors mapping
	ksFn := func(shifted, canZero bool) xproto.Keysym {
		ks := ks1
		if shifted {
			ks = ks2
		}
		if ks == 0 && !canZero {
			return max(ks1, ks2)
		}
		return ks
	}

	if isNumLockKeypad(ks1) || isNumLockKeypad(ks2) {
		if hasNumLock {
			return ksFn(!hasShift, true)
		}
		return ksFn(false, true) // no affect from shift
	} else {
		if unicode.IsLetter(keysymRune(ks1)) { // trying not to have capslock affect digits and others
			shifted := hasShift != hasCapsLock
			return ksFn(shifted, false) // no zeros, ensures a key if present; ex: downarrow can be a letter here, and if it can zero then the downarrow will not work with the shift on
		}
	}

	return ksFn(hasShift, false) // no zeros, ensures a key if present
}

//----------

func (km *KMap) modifiersToEventModifiers(m uint16) event.KeyModifiers {
	em := event.KeyModifiers(0)

	add := func(m2 uint16, em2 event.KeyModifiers) {
		if m2 != 0 && m&m2 > 0 {
			em |= em2
		}
	}

	add(km.mmask.shift, event.ModShift)
	add(km.mmask.capsL, event.ModCapsLock)
	add(km.mmask.ctrl, event.ModCtrl)
	add(km.mmask.numL, event.ModNumLock)
	add(km.mmask.alt, event.ModAlt)
	add(km.mmask.altGr, event.ModAltGr)
	add(km.mmask.superMeta, event.ModSuperMeta)

	return em
}

//----------

func (km *KMap) Dump1() string {
	o := "keysym table\n"
	for j := 0; j < 256; j++ {
		kc := xproto.Keycode(j)
		kss := km.kbm[kc]
		u := []string{}
		for _, ks := range kss {
			eks := keysymToEventKeysym(ks)
			ru := finalKeysymRune(eks, ks)
			u = append(u, fmt.Sprintf("\t(0x%x,%c,%v)", ks, ru, eks))
		}
		us := strings.Join(u, "\n")
		if len(us) > 0 {
			us = "\n" + us
		}
		o += fmt.Sprintf("kc=0x%x:%v\n", kc, us)
	}
	return o
}

func (km *KMap) Dump2() string {
	b := &bytes.Buffer{}
	pf := func(f string, args ...any) {
		fmt.Fprintf(b, f, args...)
	}

	pf("keyboard mapping (hex)\n")
	for j := 0; j < 256; j++ {
		kc := xproto.Keycode(j)
		kss := km.kbm[kc]
		u := []string{}
		for _, ks := range kss {
			u = append(u, fmt.Sprintf("0x%x", ks))
		}
		//pf("kc=%d: %v\n", j, strings.Join(u, " "))
		pf("kc=%d: {%v}\n", j, strings.Join(u, ","))
	}

	//----------

	modMap, err := xproto.GetModifierMapping(km.conn).Reply()
	if err != nil {
		panic(err)
	}

	pf("modifier mapping (hex)\n")
	stride := int8(modMap.KeycodesPerModifier)
	for g := int8(0); g < 8; g++ {
		kcs := modMap.Keycodes[g*stride : (g+1)*stride]
		u := []string{}
		for _, kc := range kcs {
			u = append(u, fmt.Sprintf("0x%x", kc))
		}
		pf("g=%d: {%v}\n", g, strings.Join(u, ","))
	}

	return b.String()
}
