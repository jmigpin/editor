package xinput

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// $ man keymaps
// https://tronche.com/gui/x/xlib/input/XGetKeyboardMapping.html

// TODO: verify modifiers column rules
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

	// log.Printf(km.KeysymTable())

	return nil
}
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
func (km *KMap) Keysym(keycode xproto.Keycode, column int) xproto.Keysym {
	x := column
	y := int(keycode - km.si.MinKeycode)
	width := int(km.reply.KeysymsPerKeycode) // usually ~7
	return km.reply.Keysyms[y*width+x]
}
func (km *KMap) modifiersColumn(mods Modifiers) int {
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

func (km *KMap) Lookup(keycode xproto.Keycode, mods Modifiers) (rune, event.KeyCode) {
	col := km.modifiersColumn(mods)
	ks := km.Keysym(keycode, col)
	ru := rune(ks)

	// extract code from the unshifted keysym (first column)
	ks0 := km.Keysym(keycode, 0)
	ru0 := rune(ks0)
	code, ok := runeKeyCodeMap[ru0]
	if !ok {
		// codes with recognizable ascii code
		switch ks0 {
		default:
			code = event.KeyCode(ks0)
		}
	}

	// runes the keysym table is not providing directly
	switch ru {
	case xkTilde:
		ru = '~'
	case xkCircumflex:
		ru = '^'
	case xkAcute:
		ru = '´'
	case xkGrave:
		ru = '`'
	}

	return ru, code
}

// Constants from /usr/include/X11/keysymdef.h
const (
	xkISOLevel3Shift = 0xfe03 // alt gr
	xkISOLeftTab     = 0xfe20

	xkBackSpace = 0xff08
	xkReturn    = 0xff0d
	xkEscape    = 0xff1b
	xkMultiKey  = 0xff20
	xkHome      = 0xff50
	xkLeft      = 0xff51
	xkUp        = 0xff52
	xkRight     = 0xff53
	xkDown      = 0xff54
	xkPageUp    = 0xff55
	xkPageDown  = 0xff56
	xkEnd       = 0xff57
	xkInsert    = 0xff63
	xkMenu      = 0xff67
	xkNumLock   = 0xff7f
	xkF1        = 0xffbe
	xkF2        = 0xffbf
	xkF3        = 0xffc0
	xkF4        = 0xffc1
	xkF5        = 0xffc2
	xkF6        = 0xffc3
	xkF7        = 0xffc4
	xkF8        = 0xffc5
	xkF9        = 0xffc6
	xkF10       = 0xffc7
	xkF11       = 0xffc8
	xkF12       = 0xffc9
	xkShiftL    = 0xffe1
	xkShiftR    = 0xffe2
	xkControlL  = 0xffe3
	xkControlR  = 0xffe4
	xkCapsLock  = 0xffe5
	xkMetaLeft  = 0xffe7
	xkMetaRight = 0xffe8
	xkAltL      = 0xffe9
	xkAltR      = 0xffea
	xkSuperL    = 0xffeb // windows key
	xkSuperR    = 0xffec
	xkDelete    = 0xffff
	xkTab       = 0xff09

	xkGrave      = 0xfe50
	xkAcute      = 0xfe51
	xkCircumflex = 0xfe52
	xkTilde      = 0xfe53

	xf86xkAudioLowerVolume = 0x1008ff11
	xf86xkAudioMute        = 0x1008ff12
	xf86xkAudioRaiseVolume = 0x1008ff13
)

var runeKeyCodeMap = map[rune]event.KeyCode{
	xkISOLevel3Shift: event.KCodeAltGr,
	xkBackSpace:      event.KCodeBackspace,
	xkReturn:         event.KCodeReturn,
	xkEscape:         event.KCodeEscape,
	xkHome:           event.KCodeHome,
	xkLeft:           event.KCodeLeft,
	xkUp:             event.KCodeUp,
	xkRight:          event.KCodeRight,
	xkDown:           event.KCodeDown,
	xkPageUp:         event.KCodePageUp,
	xkPageDown:       event.KCodePageDown,
	xkEnd:            event.KCodeEnd,
	xkInsert:         event.KCodeInsert,
	xkF1:             event.KCodeF1,
	xkF2:             event.KCodeF2,
	xkF3:             event.KCodeF3,
	xkF4:             event.KCodeF4,
	xkF5:             event.KCodeF5,
	xkF6:             event.KCodeF6,
	xkF7:             event.KCodeF7,
	xkF8:             event.KCodeF8,
	xkF9:             event.KCodeF9,
	xkF10:            event.KCodeF10,
	xkF11:            event.KCodeF11,
	xkF12:            event.KCodeF12,
	xkShiftL:         event.KCodeShiftL,
	xkShiftR:         event.KCodeShiftR,
	xkControlL:       event.KCodeControlL,
	xkControlR:       event.KCodeControlR,
	xkAltL:           event.KCodeAltL,
	xkAltR:           event.KCodeAltR,
	xkSuperL:         event.KCodeSuperL, // windows key
	xkSuperR:         event.KCodeSuperR,
	xkDelete:         event.KCodeDelete,
	xkTab:            event.KCodeTab,

	xkNumLock:  event.KCodeNumLock,
	xkCapsLock: event.KCodeCapsLock,

	xf86xkAudioLowerVolume: event.KCodeVolumeDown,
	xf86xkAudioRaiseVolume: event.KCodeVolumeUp,
	xf86xkAudioMute:        event.KCodeMute,

	//xkISOLeftTab     : event.KCode,
	//xkMultiKey  : event.KCode,
	//xkMetaLeft:  event.KCode,
	//xkMetaRight: event.KCode,
	//xkMenu      : event.KCode,
}
