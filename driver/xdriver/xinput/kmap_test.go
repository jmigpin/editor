package xinput

import (
	"os"
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestKMapLookup1(t *testing.T) {
	testLookup(t,
		0xb, 0,
		0x32, event.KSym2, '2',
	)
}
func TestKMapLookup2(t *testing.T) {
	testLookup(t,
		0xb, xproto.KeyButMaskMod5,
		0x40, event.KSymAt, '@',
	)
}
func TestKMapLookup3(t *testing.T) {
	testLookup(t,
		0x26, 0,
		0x61, event.KSymA, 'a',
	)
}
func TestKMapLookup4(t *testing.T) {
	testLookup(t,
		0x26, xproto.KeyButMaskShift,
		0x41, event.KSymA, 'A',
	)
}
func TestKMapLookup5(t *testing.T) {
	testLookup(t,
		0x23, xproto.KeyButMaskShift,
		0xfe50, event.KSymGrave, '`',
	)
}
func TestKMapLookup6(t *testing.T) {
	testLookup(t,
		0x41, 0,
		0x20, event.KSymSpace, ' ',
	)
}
func TestKMapLookup7(t *testing.T) {
	testLookup(t,
		0x33, 0,
		0xfe53, event.KSymTilde, '~',
	)
}
func TestKMapLookup8(t *testing.T) {
	testLookup(t,
		0x4d, 0,
		0xff7f, event.KSymNumLock, 'ｿ',
	)
}
func TestKMapLookup9(t *testing.T) {
	testLookup(t,
		0x5b, 0,
		0xff9f, event.KSymKeypadDelete, 'ﾟ',
	)
}
func TestKMapLookup10(t *testing.T) {
	testLookup(t,
		0x5b, xproto.KeyButMaskMod2,
		0xffae, event.KSymKeypadDecimal, '.',
	)
}
func TestKMapLookup11(t *testing.T) {
	testLookup(t,
		0x40, 0,
		0xffe9, event.KSymAltL, '￩',
	)
}
func TestKMapLookup12(t *testing.T) {
	testLookup(t,
		0x74, xproto.KeyButMaskShift|xproto.KeyButMaskControl|xproto.KeyButMaskMod1,
		0xff54, event.KSymDown, 'ｔ',
	)
}

func TestKMapLookup13(t *testing.T) {
	// keypad add: 0x56, 0xffab -> unicode u+002b

	// alter kmap for testing exotic keyboard config
	kmap, _ := getKMap(t)
	kss := kmap.keycodeToKeysyms(0x56)
	group := 0
	i := group * 2
	ks1p := &kss[i]
	ks2p := &kss[i+1]

	*ks1p = 0xffab
	*ks2p = 0x100002b

	//fmt.Println(kmap.keysymsTableStr())

	// replace/restore global var
	tmp := gkmap
	gkmap = kmap
	defer func() { gkmap = tmp }()

	testLookup(t,
		0x56, 0,
		0xffab, event.KSymKeypadAdd, '+',
	)
	testLookup(t,
		0x56, xproto.KeyButMaskShift,
		0x100002b, event.KSymNone, '+',
	)
}

//----------
//----------
//----------

var gkmap *KMap

func testLookup(
	t *testing.T,

	kc xproto.Keycode,
	kmods uint16,

	ks2 xproto.Keysym,
	eks2 event.KeySym,
	ru2 rune,
) {
	t.Helper()
	if gkmap == nil {
		gkmap, _ = getKMap(t)

		//fmt.Println(gkmap.keysymsTableStr())
	}
	ks1, eks1, ru1 := gkmap.Lookup(kc, kmods)

	if ks1 != ks2 || eks1 != eks2 || ru1 != ru2 {
		t.Fatalf("->(0x%x,%v)\nexp:(0x%x,%v,%q)\ngot:(0x%x,%v,%q)",
			kc, kmods,
			ks2, eks2, ru2,
			ks1, eks1, ru1,
		)
	}
}

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
