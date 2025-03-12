package xinput

import (
	"os"
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestKMapLookup1(t *testing.T) {
	kmap, conn := getKMap(t)
	defer conn.Close()

	type pair struct {
		kc    xproto.Keycode
		kmods uint16

		ks  xproto.Keysym
		eks event.KeySym
		ru  rune
	}
	pairs := []pair{
		{
			0xb, 0,
			0x32, event.KSym2, '2',
		},
		{
			0xb, xproto.KeyButMaskMod5,
			0x40, event.KSymAt, '@',
		},
		{
			0x26, 0,
			0x61, event.KSymA, 'a',
		},
		{
			0x26, xproto.KeyButMaskShift,
			0x41, event.KSymA, 'A',
		},
		{
			0x23, xproto.KeyButMaskShift,
			0xfe50, event.KSymGrave, '`',
		},
		{
			0x41, 0,
			0x20, event.KSymSpace, ' ',
		},
		{
			0x33, 0,
			0xfe53, event.KSymTilde, '~',
		},
		{
			0x4d, 0,
			0xff7f, event.KSymNumLock, 'ｿ',
		},
		{
			0x5b, 0,
			0xff9f, event.KSymKeypadDelete, 'ﾟ',
		},
		{
			0x5b, xproto.KeyButMaskMod2,
			0xffae, event.KSymKeypadDecimal, '.',
		},
		{
			0x40, 0,
			0xffe9, event.KSymAltL, '￩',
		},
		{
			0x74, xproto.KeyButMaskShift | xproto.KeyButMaskControl | xproto.KeyButMaskMod1,
			0xff54, event.KSymDown, 'ｔ',
		},
	}

	//println(kmap.keysymsTableStr())

	for i, p := range pairs {
		ks, eks, ru := kmap.Lookup(p.kc, p.kmods)
		if ks != p.ks || eks != p.eks || ru != p.ru {
			t.Fatalf("entry %v:\n(0x%x,%v)->(0x%x,%v,%q)\nexpected (0x%x,%v,%q)\n",
				i,
				p.kc, p.kmods,
				ks, eks, ru,
				p.ks, p.eks, p.ru,
			)
		}
	}
}

//----------
//----------
//----------

func getKMap(t *testing.T) (*KMap, *xgb.Conn) {
	display := os.Getenv("DISPLAY")
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		t.Fatal(err)
	}
	km, err := NewKMap(conn)
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	return km, conn
}
