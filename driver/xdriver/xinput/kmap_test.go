package xinput

import (
	"fmt"
	"os"
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

// will only work under certain x11 configs
const shiftMask = uint16(xproto.KeyButMaskShift)
const capsMask = uint16(xproto.KeyButMaskLock)
const ctrlMask = uint16(xproto.KeyButMaskControl)
const altMask = uint16(xproto.KeyButMaskMod1)
const numMask = uint16(xproto.KeyButMaskMod2)
const altGrMask = uint16(xproto.KeyButMaskMod5)

func TestKMapLookup1(t *testing.T) {
	testLookup(t,
		0xb, 0,
		0x32, event.KSym2, '2',
	)
}
func TestKMapLookup2(t *testing.T) {
	testLookup(t,
		0xb, altGrMask,
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
		0x26, shiftMask,
		0x41, event.KSymA, 'A',
	)
}
func TestKMapLookup5(t *testing.T) {
	testLookup(t,
		0x23, shiftMask,
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
		0x40, 0,
		0xffe9, event.KSymAltL, '￩',
	)
}
func TestKMapLookup11(t *testing.T) {
	testLookup(t,
		//0x74, shiftMask|ctrlMask|altMask,
		0x74, shiftMask|ctrlMask,
		0xff54, event.KSymDown, 'ｔ',
	)
}
func TestKMapLookup12(t *testing.T) {
	testLookup(t,
		0x57, 0,
		0xff9c, event.KSymNone, 'ﾜ',
	)
	testLookup(t,
		0x57, numMask,
		0xffb1, event.KSymKeypad1, '1',
	)
	testLookup(t,
		// shift not affecting keypad digit
		0x57, shiftMask,
		0xff9c, event.KSymNone, 'ﾜ',
	)
	testLookup(t,
		// with numlock on, shift can affect keypad digit
		0x57, numMask|shiftMask,
		0xff9c, event.KSymNone, 'ﾜ',
	)
}

//----------

func TestKMapLookupC1(t *testing.T) {
	kc := xproto.Keycode(0x5b)
	restore := setupKmapReplacePair(t,
		kc,
		0,
		0x2e, // KSymPeriod
		0xff9f,
	)
	defer restore()

	testLookup(t,
		kc, numMask,
		0x2e, event.KSymPeriod, '.',
	)
	testLookup(t,
		kc, numMask|shiftMask,
		0xff9f, event.KSymKeypadDelete, 'ﾟ',
	)
}

func TestKMapLookupC2(t *testing.T) {
	kc := xproto.Keycode(0x56) // 86
	restore := setupKmapReplacePair(t,
		kc,
		0,
		0xffab,    // KSymKeypadAdd
		0x100002b, // u+002b
	)
	defer restore()

	testLookup(t,
		kc, numMask,
		0xffab, event.KSymKeypadAdd, '+',
	)
	testLookup(t,
		kc, numMask|shiftMask,
		0x100002b, event.KSymNone, '+',
	)
}

func TestKMapLookupC3(t *testing.T) {
	kc := xproto.Keycode(0x3f) // 63
	restore := setupKmapReplacePair(t,
		kc,
		0,
		0xffaa, // KSymKeypadMultiply
		0x10022c5,
	)
	defer restore()

	// numlock not affecting "*" multiply, but shift does
	testLookup(t,
		kc, 0,
		0xffaa, event.KSymKeypadMultiply, '*',
	)
	testLookup(t,
		kc, numMask,
		0xffaa, event.KSymKeypadMultiply, '*',
	)
	testLookup(t,
		kc, numMask|shiftMask,
		0x10022c5, event.KSymNone, '⋅',
	)
	testLookup(t,
		kc, shiftMask,
		0x10022c5, event.KSymNone, '⋅',
	)
}

// TODO: keypad divide "/" "∕"?

//----------

func TestDumpMapping(t *testing.T) {
	// comment this to enable
	t.Skip("avoid dumping in general tests")

	km, _ := getKMap(t)
	fmt.Print(km.Dump2())
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

	//fmt.Println(km.keysymsTableStr())

	return km, conn
}

func setupKmapReplacePair(t *testing.T, kc xproto.Keycode, group int, ks1, ks2 xproto.Keysym) func() {
	kmap, _ := getKMap(t)
	// replace/restore global var
	tmp := gkmap
	gkmap = kmap
	//defer func() { gkmap = tmp }()

	// alter kmap for testing exotic keyboard config
	kss := kmap.keycodeToKeysyms(kc)
	i := group * 2
	ks1p := &kss[i]
	ks2p := &kss[i+1]

	*ks1p = ks1
	*ks2p = ks2

	return func() {
		gkmap = tmp
	}
}
