package keybmap

import (
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// $ man keymaps
// https://tronche.com/gui/x/xlib/input/XGetKeyboardMapping.html

// TODO: update map in case of key map change (mapping notify)

type KeybMap struct {
	conn      *xgb.Conn
	setupInfo *xproto.SetupInfo
	keybMap   *xproto.GetKeyboardMappingReply
	modMap    *xproto.GetModifierMappingReply
}

func NewKeybMap(conn *xgb.Conn, si *xproto.SetupInfo) (*KeybMap, error) {
	km := &KeybMap{conn: conn, setupInfo: si}

	// get keyboard mapping
	count := byte(km.setupInfo.MaxKeycode - km.setupInfo.MinKeycode + 1)
	cookie := xproto.GetKeyboardMapping(km.conn, km.setupInfo.MinKeycode, count)
	reply, err := cookie.Reply()
	if err != nil {
		return nil, err
	}
	km.keybMap = reply
	//fmt.Printf("%+#v\n",reply)

	//// debug
	//min:=km.setupInfo.MinKeycode
	//max:=km.setupInfo.MaxKeycode
	//for i:=min;i<max;i++{
	//km.debug(i,0)
	//}

	//// debug
	//fmt.Printf("min: %d\n",km.setupInfo.MinKeycode)
	//for i,ks:=range km.keybMap.Keysyms{
	//fmt.Printf("%d: ('%c',%x)\n",i, ks,ks)
	//}

	//fmt.Printf("min %v, width %v\n",km.setupInfo.MinKeycode, km.keybMap.KeysymsPerKeycode)

	// get modifier mapping
	// TODO: if buttons/keys modifiers get remapped
	cookie2 := xproto.GetModifierMapping(km.conn)
	reply2, err := cookie2.Reply()
	if err != nil {
		return nil, err
	}
	km.modMap = reply2
	//fmt.Printf("%+#v\n",reply2)

	return km, nil
}

func (km *KeybMap) KeysymColumn(keycode xproto.Keycode, column int) xproto.Keysym {
	kc := int(keycode - km.setupInfo.MinKeycode)
	w := int(km.keybMap.KeysymsPerKeycode)
	i := kc*w + column
	return km.keybMap.Keysyms[i]
}

func (km *KeybMap) debug(keycode xproto.Keycode, mods Modifiers) {
	fmt.Printf("*kb: m=%x, kc=%x, syms:", mods, keycode)
	w := int(km.keybMap.KeysymsPerKeycode) // ~7
	for j := 0; j < w; j++ {
		u := km.KeysymColumn(keycode, j)
		fmt.Printf("('%c',%x)", u, u)
	}
	fmt.Printf("\n")
}

func (km *KeybMap) ModKeysym(keycode xproto.Keycode, mods Modifiers) xproto.Keysym {
	col := km.modifiersColumn(mods)
	return km.KeysymColumn(keycode, col)
}
func (km *KeybMap) modifiersColumn(mods Modifiers) int {
	//alt := mods.Mod1()
	altGr := mods.Mod5()

	caps := mods.CapsLock()
	shift := mods.Shift()
	shift = (shift && !caps) || (!shift && caps)

	ctrl := mods.Control()

	// missing: 3,6
	i := 0
	if altGr {
		i = 4
		if shift {
			i = 5
		}
	} else if shift {
		i = 1
	} else if ctrl {
		i = 2
	}

	if i >= int(km.keybMap.KeysymsPerKeycode) {
		panic("i>=keysymsperkeycode")
	}

	return i
}
