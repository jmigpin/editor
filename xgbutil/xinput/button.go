package xinput

import "github.com/BurntSushi/xgb/xproto"

type Button struct {
	km     *KMap
	button xproto.Button // xproto.ButtonIndex1...
	Mods   Modifiers     // xproto.KeyButMaskButton1...
}

func NewButton(km *KMap, button xproto.Button, state uint16) *Button {

	// TODO: keypress mods differ from keyrelease

	return &Button{km, button, Modifiers(state)}
}
func (b *Button) Button(v int) bool {
	if v >= 0 && v <= 5 {
		// xproto.ButtonIndex1 = 1
		return v == int(b.button)
	}
	panic("button index out of range")
}
