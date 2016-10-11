package keybmap

import "github.com/BurntSushi/xgb/xproto"

type Modifiers uint16 // key and button mask

func (m Modifiers) On(v uint) bool {
	return uint(m)&v > 0
}

func (m Modifiers) Shift() bool    { return m.On(xproto.KeyButMaskShift) }
func (m Modifiers) CapsLock() bool { return m.On(xproto.KeyButMaskLock) }
func (m Modifiers) Control() bool  { return m.On(xproto.KeyButMaskControl) }
func (m Modifiers) Mod1() bool     { return m.On(xproto.KeyButMaskMod1) } // Alt
func (m Modifiers) Mod2() bool     { return m.On(xproto.KeyButMaskMod2) }
func (m Modifiers) Mod3() bool     { return m.On(xproto.KeyButMaskMod3) }    // Num lock
func (m Modifiers) Mod4() bool     { return m.On(xproto.KeyButMaskMod4) }    // Windows key
func (m Modifiers) Mod5() bool     { return m.On(xproto.KeyButMaskMod5) }    // AltGr
func (m Modifiers) Button1() bool  { return m.On(xproto.KeyButMaskButton1) } // left
func (m Modifiers) Button2() bool  { return m.On(xproto.KeyButMaskButton2) } // middle
func (m Modifiers) Button3() bool  { return m.On(xproto.KeyButMaskButton3) } // right
func (m Modifiers) Button4() bool  { return m.On(xproto.KeyButMaskButton4) } // wheel up
func (m Modifiers) Button5() bool  { return m.On(xproto.KeyButMaskButton5) } // wheel down
