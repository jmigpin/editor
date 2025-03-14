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
	si    *xproto.SetupInfo
	reply *xproto.GetKeyboardMappingReply
	conn  *xgb.Conn

	modGroups struct {
		numLock int8
		alt     int8
		altGr   int8
		super   int8
		meta    int8
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

	//log.Printf("%v", km.keysymsTableStr())

	return nil
}

func (km *KMap) readModMapping() error {
	modMap, err := xproto.GetModifierMapping(km.conn).Reply()
	if err != nil {
		return err
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

	// X11: keysyms to detect which group might have them
	type KS = xproto.Keysym
	numLocks := []KS{
		0xff7f, // XK_Num_Lock
	}
	alts := []KS{
		0xffe9, // XK_Alt_L
		0xffea, // XK_Alt_R
	}
	altGrs := []KS{
		0xfe03, // XK_ISO_Level3_Shift
		0xfe11, // XK_ISO_Level5_Shift
		0xff7e, // XK_ISO_Group_Shift
	}
	supers := []KS{
		0xffeb, // XK_Super_L
		0xffec, // XK_Super_R
	}
	metas := []KS{
		0xffe7, // XK_Meta_L
		0xffe8, // XK_Meta_R
	}

	// defaults
	km.modGroups.numLock = 4
	km.modGroups.alt = 3
	km.modGroups.altGr = 7
	km.modGroups.super = -1
	km.modGroups.meta = -1

	type pair struct {
		group *int8
		kss   []KS
	}

	pairs := []pair{
		pair{&km.modGroups.numLock, numLocks},
		pair{&km.modGroups.alt, alts},
		pair{&km.modGroups.altGr, altGrs},
		pair{&km.modGroups.super, supers},
		pair{&km.modGroups.meta, metas},
	}
	_ = metas

	// detect
	stride := int8(modMap.KeycodesPerModifier)
	for g := int8(3); g < 8; g++ {
		kcs := modMap.Keycodes[g*stride : (g+1)*stride]
		//fmt.Println(g, kcs) // DEBUG
	kcLoop: // iterate keycodes/keysyms, keep first found group
		for _, kc := range kcs { //
			kss := km.keycodeToKeysyms(kc)
			//fmt.Println("\t", kss) // DEBUG
			for _, ks := range kss {
				for pi, p := range pairs {
					_ = pi
					for _, ks2 := range p.kss {
						if ks == ks2 {
							*p.group = g
							//fmt.Println("detected", g, "for pair", pi, *p.group, fmt.Sprintf("%b", *p.group)) // DEBUG
							break kcLoop
						}
					}
				}
			}
		}
	}

	return nil
}

//----------

func (km *KMap) Lookup(keycode xproto.Keycode, kmods uint16) (xproto.Keysym, event.KeySym, rune) {
	kss := km.keycodeToKeysyms(keycode)
	ks := km.keysymsToKeysym(kss, kmods)
	eks := keysymToEventKeysym(ks)
	ru := finalKeysymRune(eks, ks)
	return ks, eks, ru
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

//func (km *KMap) keysymsToKeysym(kss []xproto.Keysym, m uint16) xproto.Keysym {
//	em := km.modifiersToEventModifiers(m)

//	hasShift := em.HasAny(event.ModShift)
//	hasCtrl := em.HasAny(event.ModCtrl)
//	hasAltGr := em.HasAny(event.ModAltGr)
//	hasCapsLock := em.HasAny(event.ModCapsLock)
//	hasNumLock := em.HasAny(event.ModNumLock)

//	// keysym group
//	group := 0
//	if hasAltGr {
//		group = 2
//		if hasCtrl { // TODO: this is custom, since ctrl alone is probably not the correct group changer
//			group = 1
//		}
//	}

//	// each group has two symbols (normal and shifted)
//	i1 := group * 2
//	i2 := i1 + 1
//	if i1 >= len(kss) {
//		return 0
//	}
//	if i2 >= len(kss) {
//		i2 = i1
//	}
//	ks1, ks2 := kss[i1], kss[i2]

//	// only one is valid, return the other // TODO: this is custom since some combinations are not supposed to return anything - this simplifies and gives access to the defined key
//	if ks1 == 0 {
//		return ks2
//	}
//	if ks2 == 0 {
//		return ks1
//	}

//	// have both ks1 and ks2: determine shift
//	shifted := hasShift
//	if isKeypad(ks1) { // TODO: hopefully, ks2 is in keypad too!
//		if hasNumLock {
//			shifted = !shifted
//		}
//	} else {
//		if hasCapsLock {
//			shifted = !shifted
//		}
//	}

//	if shifted {
//		return ks2
//	}
//	return ks1
//}

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

	if isKeypadDigit(ks1) || isKeypadDigit(ks2) {
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
	addGroup := func(g int8, em2 event.KeyModifiers) {
		if g < 0 { // not detected
			return
		}
		add(1<<g, em2)
	}

	add(xproto.KeyButMaskShift, event.ModShift)
	add(xproto.KeyButMaskLock, event.ModCapsLock)
	add(xproto.KeyButMaskControl, event.ModCtrl)

	addGroup(km.modGroups.numLock, event.ModNumLock)
	addGroup(km.modGroups.alt, event.ModAlt)
	addGroup(km.modGroups.altGr, event.ModAltGr)
	addGroup(km.modGroups.super, event.ModSuper)
	addGroup(km.modGroups.meta, event.ModMeta)

	return em
}

//----------

func (km *KMap) Dump1() string {
	o := "keysym table\n"
	for j := 0; j < 256; j++ {
		kc := xproto.Keycode(j)
		kss := km.keycodeToKeysyms(kc)
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
		kss := km.keycodeToKeysyms(kc)
		u := []string{}
		for _, ks := range kss {
			u = append(u, fmt.Sprintf("%x", ks))
		}
		pf("kc=%d: %v\n", j, strings.Join(u, " "))
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
			u = append(u, fmt.Sprintf("%x", kc))
		}
		pf("g=%d: %v\n", g, strings.Join(u, " "))
	}

	return b.String()
}
