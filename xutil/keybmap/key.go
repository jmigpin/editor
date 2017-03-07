package keybmap

import "github.com/BurntSushi/xgb/xproto"

type Key struct {
	km        *KeybMap
	Keycode   xproto.Keycode // byte
	Modifiers Modifiers
}

func newKey(km *KeybMap, keycode xproto.Keycode, state uint16) *Key {
	return &Key{km, keycode, Modifiers(state)}
}
func (k *Key) FirstKeysym() xproto.Keysym {
	return k.km.KeysymColumn(k.Keycode, 0)
}
func (k *Key) Keysym() xproto.Keysym {
	return k.km.ModKeysym(k.Keycode, k.Modifiers)
}
