package dragndrop

import "github.com/BurntSushi/xgb/xproto"

type EnterEvent struct {
	Window             xproto.Window
	MoreThan3DataTypes bool
	ProtoVersion       int
	Types              []xproto.Atom
}

func ParseEnterEvent(buf []uint32) *EnterEvent {
	return &EnterEvent{
		Window:             xproto.Window(buf[0]),
		MoreThan3DataTypes: buf[1]&1 == 1,
		ProtoVersion:       int(buf[1] >> 24),
		Types: []xproto.Atom{
			xproto.Atom(buf[2]),
			xproto.Atom(buf[3]),
			xproto.Atom(buf[4]),
		},
	}
}
func (enter *EnterEvent) SupportsType(typ xproto.Atom) bool {
	for _, t := range enter.Types {
		if typ == t {
			return true
		}
	}
	return false
}
