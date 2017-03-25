package keybmap

import "github.com/BurntSushi/xgb/xproto"

type Button struct {
	km     *KeybMap
	Button xproto.Button // xproto.ButtonIndex1...
	Mods   Modifiers     // xproto.KeyButMaskButton1...
}

func newButton(km *KeybMap, button xproto.Button, state uint16) *Button {

	// TODO: keypress mods differ from keyrelease

	return &Button{km, button, Modifiers(state)}
}
func (b *Button) Button1() bool {
	return b.Button == xproto.ButtonIndex1
}
func (b *Button) Button2() bool {
	return b.Button == xproto.ButtonIndex2
}
func (b *Button) Button3() bool {
	return b.Button == xproto.ButtonIndex3
}
func (b *Button) Button4() bool {
	return b.Button == xproto.ButtonIndex4
}
func (b *Button) Button5() bool {
	return b.Button == xproto.ButtonIndex5
}
