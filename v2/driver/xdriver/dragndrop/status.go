package dragndrop

import "github.com/BurntSushi/xgb/xproto"

type StatusEvent struct {
	Window xproto.Window
	Flags  uint32
	Action xproto.Atom
}

func (st *StatusEvent) Data32() []uint32 {
	return []uint32{
		uint32(st.Window),
		st.Flags,
		0,                 // x,y
		0,                 // w,h
		uint32(st.Action), // accepted action
	}
}

const (
	StatusEventAcceptFlag        = 1 << 0
	StatusEventSendPositionsFlag = 1 << 1 // ask to keep sending positions
)
