package xinput

import (
	"fmt"
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// $ man keymaps
// https://tronche.com/gui/x/xlib/input/XGetKeyboardMapping.html

// TODO: update map in case of key map change (mapping notify)

// Keyboard mapping
type KMap struct {
	si  *xproto.SetupInfo
	kmr *xproto.GetKeyboardMappingReply
}

func NewKMap(conn *xgb.Conn) (*KMap, error) {
	si := xproto.Setup(conn)

	count := byte(si.MaxKeycode - si.MinKeycode + 1)
	cookie := xproto.GetKeyboardMapping(conn, si.MinKeycode, count)
	reply, err := cookie.Reply()
	if err != nil {
		return nil, err
	}

	km := &KMap{si: si, kmr: reply}
	return km, nil
}
func (km *KMap) KeysymColumn(keycode xproto.Keycode, column int) xproto.Keysym {
	kc := int(keycode - km.si.MinKeycode)
	w := int(km.kmr.KeysymsPerKeycode)
	i := kc*w + column
	return km.kmr.Keysyms[i]
}
func (km *KMap) modifiersColumn(mods Modifiers) int {
	// TODO: rules
	// https://tronche.com/gui/x/xlib/input/keyboard-encoding.html

	altGr := xproto.KeyButMaskMod5
	shift := xproto.KeyButMaskShift
	caps := xproto.KeyButMaskLock
	ctrl := xproto.KeyButMaskControl

	// missing: 3,6
	i := 0
	switch {
	case mods.Is(altGr):
		i = 4
	case mods.Is(altGr|shift) || mods.Is(altGr|caps):
		i = 5
	case mods.Is(ctrl):
		i = 2
	case mods.Is(shift) || mods.Is(caps):
		i = 1
	}
	return i
}

func (km *KMap) debug(keycode xproto.Keycode, mods Modifiers) {
	s := fmt.Sprintf("*kb: m=%x, kc=%x, syms:", mods, keycode)
	w := int(km.kmr.KeysymsPerKeycode) // ~7
	for j := 0; j < w; j++ {
		u := km.KeysymColumn(keycode, j)
		s += fmt.Sprintf("('%c',%x)", u, u)
	}
	s += "\n"
	log.Print(s)
}
