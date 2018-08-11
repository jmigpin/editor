package xinput

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// $ man keymaps
// https://tronche.com/gui/x/xlib/input/XGetKeyboardMapping.html
// https://tronche.com/gui/x/xlib/input/keyboard-encoding.html
// http://wiki.linuxquestions.org/wiki/List_of_Keysyms_Recognised_by_Xmodmap
// https://www.x.org/releases/X11R7.7/doc/libX11/i18n/compose/iso8859-2.html

// Keyboard mapping
type KMap struct {
	si    *xproto.SetupInfo
	reply *xproto.GetKeyboardMappingReply
	conn  *xgb.Conn
}

func NewKMap(conn *xgb.Conn) (*KMap, error) {
	km := &KMap{conn: conn}
	err := km.ReadTable()
	if err != nil {
		return nil, err
	}
	return km, nil
}

//----------

func (km *KMap) ReadTable() error {
	si := xproto.Setup(km.conn)
	count := byte(si.MaxKeycode - si.MinKeycode + 1)
	if count == 0 {
		return fmt.Errorf("count is 0")
	}
	cookie := xproto.GetKeyboardMapping(km.conn, si.MinKeycode, count)
	reply, err := cookie.Reply()
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

//----------

func (km *KMap) KeysymTable() string {
	// some symbols are not present, like "~" and "^", and their X11 constant is present instead
	o := "keysym table\n"
	table := km.reply.Keysyms
	width := int(km.reply.KeysymsPerKeycode)
	for y := 0; y*width < len(table); y++ {
		var u []string
		for x := 0; x < width; x++ {
			u = append(u, fmt.Sprintf("%c", rune(table[y*width+x])))
		}
		o += fmt.Sprintf("%v: %v\n", y*width, strings.Join(u, ", "))
	}
	return o
}

//----------

func (km *KMap) colKeysym(keycode xproto.Keycode, column int) xproto.Keysym {
	x := column
	y := int(keycode - km.si.MinKeycode)
	width := int(km.reply.KeysymsPerKeycode) // usually ~7
	return km.reply.Keysyms[y*width+x]
}

func (km *KMap) keysymRow(keycode xproto.Keycode) []xproto.Keysym {
	y := int(keycode - km.si.MinKeycode)
	width := int(km.reply.KeysymsPerKeycode)
	return km.reply.Keysyms[y*width : y*width+width]
}

//----------

func (km *KMap) printKeysyms(keycode xproto.Keycode) {
	keysyms := km.keysymRow(keycode)
	//fmt.Printf("%v\n", keysyms)

	{
		u := []string{}
		for _, ks := range keysyms {
			u = append(u, string(ks))
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

func isKeypad(ks xproto.Keysym) bool {
	return (0xFF80 <= ks && ks <= 0xFFBD) ||
		(0x11000000 <= ks && ks <= 0x1100FFFF)
}

//----------

// xproto.Keycode is a physical key.
// xproto.Keysym is the encoding of a symbol on the cap of a key.
// A list of keysyms is associated with each keycode.

func (km *KMap) keysym(keycode xproto.Keycode, m Modifiers) xproto.Keysym {
	hasShift := m.HasAny(xproto.KeyButMaskShift)
	hasCaps := m.HasAny(xproto.KeyButMaskLock)
	hasCtrl := m.HasAny(xproto.KeyButMaskControl)
	hasNum := m.HasAny(xproto.KeyButMaskMod2)
	hasAltGr := m.HasAny(xproto.KeyButMaskMod5)

	krow := km.keysymRow(keycode)

	//// DEBUG
	//log.Printf("kc=%v %b, m=%v %b", keycode, keycode, m, m)
	//km.printKeysyms(keycode)

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
	if i1 >= len(krow) {
		return 0
	}
	if i2 >= len(krow) {
		i2 = i1
	}
	ks1, ks2 := krow[i1], krow[i2]
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

	//log.Printf("iskeypad=%v", isKeypad(ks2))

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

func (km *KMap) Lookup(keycode xproto.Keycode, mods Modifiers) (rune, event.KeySym) {
	xks := km.keysym(keycode, mods)
	ks, ok := xkeysymToEvKeySym[xks]
	if !ok {
		// will keep ascii codes and others ('a', 'A', ...)
		ks = event.KeySym(xks)
	}
	ru := keysymRune(xks)
	return ru, ks
}

//----------

func keysymRune(ks xproto.Keysym) rune {
	// runes the keysym table is not providing directly
	switch ks {
	case xkTilde:
		return '~'
	case xkCircumflex:
		return '^'
	case xkAcute:
		return '´'
	case xkGrave:
		return '`'

	case xkKeypadDecimal:
		return '.'
	case xkKeypadAdd:
		return '+'
	case xkKeypadSubtract:
		return '-'
	case xkKeypadMultiply:
		return '*'
	case xkKeypadDivide:
		return '/'

	case xkKeypad0:
		return '0'
	case xkKeypad1:
		return '1'
	case xkKeypad2:
		return '2'
	case xkKeypad3:
		return '3'
	case xkKeypad4:
		return '4'
	case xkKeypad5:
		return '5'
	case xkKeypad6:
		return '6'
	case xkKeypad7:
		return '7'
	case xkKeypad8:
		return '8'
	case xkKeypad9:
		return '9'
	}
	return rune(ks)
}
