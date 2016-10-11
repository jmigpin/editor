package dragndrop

import "github.com/BurntSushi/xgb/xproto"

type FinishedEvent struct {
	Window   xproto.Window
	Accepted bool
	Action   xproto.Atom
}

func (f *FinishedEvent) Data32() []uint32 {
	acc := uint32(0)
	if f.Accepted {
		acc = 1 // bit 0
	}
	//} else {
	// if not accepted, this should also be none
	//f.Action = xproto.AtomNone
	//}
	return []uint32{
		uint32(f.Window),
		acc,
		uint32(f.Action),
		uint32(0), // pad
		uint32(0), // pad
	}
}
