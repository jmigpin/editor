package xinput

import (
	"fmt"
	"io"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// XKB wire layouts follow XKBproto.h: UseExtension, GetMap, key types and symbol maps.
type xkbMap struct {
	conn   *xgb.Conn
	opcode byte
	types  []xkbKeyType
	keys   [256]xkbKey
}

func newXKBMap(conn *xgb.Conn) (*xkbMap, error) {
	const name = "XKEYBOARD"
	ext, err := xproto.QueryExtension(conn, uint16(len(name)), name).Reply()
	if err != nil {
		return nil, err
	}
	if !ext.Present {
		return nil, nil
	}
	m := &xkbMap{conn: conn, opcode: ext.MajorOpcode}
	buf := make([]byte, 8)
	buf[0] = m.opcode
	xgb.Put16(buf[2:], 2)
	xgb.Put16(buf[4:], 1)
	cookie := conn.NewCookie(true, true)
	conn.NewRequest(buf, cookie)
	reply, err := cookie.Reply()
	if err != nil {
		return nil, err
	}
	if len(reply) < 32 {
		return nil, io.ErrUnexpectedEOF
	}
	if reply[1] == 0 {
		return nil, nil
	}
	// Route XKB map/replacement notifications through the existing MappingNotify reload path.
	if _, ok := xgb.NewEventFuncs[int(ext.FirstEvent)]; !ok {
		xgb.NewEventFuncs[int(ext.FirstEvent)] = func(buf []byte) xgb.Event {
			return xproto.MappingNotifyEvent{Sequence: xgb.Get16(buf[2:]), Request: xproto.MappingKeyboard}
		}
	}
	buf = make([]byte, 16)
	buf[0], buf[1] = m.opcode, 1 // SelectEvents
	xgb.Put16(buf[2:], 4)
	xgb.Put16(buf[4:], 0x100)
	xgb.Put16(buf[6:], 3)  // NewKeyboardNotify | MapNotify
	xgb.Put16(buf[10:], 3) // Select all details for these events.
	xgb.Put16(buf[12:], 3) // KeyTypes | KeySyms
	xgb.Put16(buf[14:], 3)
	cookie = conn.NewCookie(true, false)
	conn.NewRequest(buf, cookie)
	if err := cookie.Check(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *xkbMap) readMapping() error {
	buf := make([]byte, 28)
	buf[0], buf[1] = m.opcode, 8 // GetMap
	xgb.Put16(buf[2:], 7)
	xgb.Put16(buf[4:], 0x100) // UseCoreKbd
	xgb.Put16(buf[6:], 3)     // KeyTypes | KeySyms; resolved modifier masks are included in the reply.
	cookie := m.conn.NewCookie(true, true)
	m.conn.NewRequest(buf, cookie)
	reply, err := cookie.Reply()
	if err != nil {
		return err
	}
	return m.decodeMapping(reply)
}

func (m *xkbMap) decodeMapping(reply []byte) error {
	if len(reply) < 40 {
		return io.ErrUnexpectedEOF
	}
	if xgb.Get16(reply[12:]) != 3 || reply[14] != 0 {
		return fmt.Errorf("unexpected XKB map components or type range")
	}
	r := &xkbReader{buf: reply[40:]}
	types := make([]xkbKeyType, int(reply[15]))
	for i := range types {
		h := r.take(8)
		if r.err != nil {
			return r.err
		}
		kt := &types[i]
		kt.mask = h[0]
		n, preserve := int(h[5]), h[6] != 0
		for range n {
			e := r.take(8)
			if r.err != nil {
				return r.err
			}
			if e[0] != 0 {
				kt.entries = append(kt.entries, xkbTypeEntry{mask: e[1], level: e[2]})
			}
		}
		if preserve {
			r.take(n * 4)
		}
	}
	keys := [256]xkbKey{}
	first, count := int(reply[17]), int(reply[20])
	if first+count > len(keys) {
		return fmt.Errorf("invalid XKB key range")
	}
	for i := first; i < first+count; i++ {
		h := r.take(8)
		if r.err != nil {
			return r.err
		}
		k := &keys[i]
		copy(k.types[:], h[:4])
		k.groupInfo, k.width = h[4], h[5]
		n := int(xgb.Get16(h[6:]))
		for group := 0; group < int(k.groupInfo&15); group++ {
			if group >= len(k.types) || int(k.types[group]) >= len(types) {
				return fmt.Errorf("invalid XKB group or type index")
			}
		}
		bs := r.take(n * 4)
		if r.err != nil {
			return r.err
		}
		k.symbols = make([]xproto.Keysym, n)
		for j := range k.symbols {
			k.symbols[j] = xproto.Keysym(xgb.Get32(bs[j*4:]))
		}
	}
	if r.err != nil {
		return r.err
	}
	m.types, m.keys = types, keys
	return nil
}

func (m *xkbMap) lookup(kc xproto.Keycode, state uint16) xproto.Keysym {
	k := &m.keys[kc]
	nGroups := int(k.groupInfo & 15)
	if nGroups == 0 {
		return 0
	}
	group := int(state>>13) & 3
	if group >= nGroups {
		switch k.groupInfo & 0xc0 {
		case 0x40: // ClampIntoRange
			group = nGroups - 1
		case 0x80: // RedirectIntoRange
			group = int(k.groupInfo>>4) & 3
			if group >= nGroups {
				group = 0
			}
		default:
			group %= nGroups
		}
	}
	kt := &m.types[k.types[group]]
	mods := byte(state) & kt.mask
	level := 0
	for _, e := range kt.entries {
		if mods == e.mask {
			level = int(e.level)
			break
		}
	}
	i := group*int(k.width) + level
	if level >= int(k.width) || i >= len(k.symbols) {
		return 0
	}
	return k.symbols[i]
}

//----------

type xkbKeyType struct {
	mask    byte
	entries []xkbTypeEntry
}

//----------

type xkbTypeEntry struct {
	mask, level byte
}

//----------

type xkbKey struct {
	types     [4]byte
	groupInfo byte
	width     byte
	symbols   []xproto.Keysym
}

//----------

type xkbReader struct {
	buf []byte
	err error
}

func (r *xkbReader) take(n int) []byte {
	if r.err != nil || n > len(r.buf) {
		r.err = io.ErrUnexpectedEOF
		return nil
	}
	b := r.buf[:n]
	r.buf = r.buf[n:]
	return b
}
