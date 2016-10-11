package keybmap

import "github.com/BurntSushi/xgb/xproto"

type Button struct {
	km     *KeybMap
	Button xproto.Button // byte
	Mods   Modifiers
}

func NewButton(km *KeybMap, button xproto.Button, state uint16) *Button {
	return &Button{km, button, Modifiers(state)}
}
func (b *Button) Button1() bool { return b.Button == xproto.ButtonIndex1 }
func (b *Button) Button2() bool { return b.Button == xproto.ButtonIndex2 }
func (b *Button) Button3() bool { return b.Button == xproto.ButtonIndex3 }
func (b *Button) Button4() bool { return b.Button == xproto.ButtonIndex4 }
func (b *Button) Button5() bool { return b.Button == xproto.ButtonIndex5 }
