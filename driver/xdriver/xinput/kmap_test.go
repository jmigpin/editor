package xinput

import (
	"fmt"
	"os"
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestKMapLookup1(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0xb, 0,
		0x32, event.KSym2, '2',
	)
}
func TestKMapLookup2(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0xb, km.mmask.altGr,
		0x40, event.KSymAt, '@',
	)
}
func TestKMapLookup3(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x26, 0,
		0x61, event.KSymA, 'a',
	)
}
func TestKMapLookup4(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x26, km.mmask.shift,
		0x41, event.KSymA, 'A',
	)
}
func TestKMapLookup5(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x23, km.mmask.shift,
		0xfe50, event.KSymGrave, '`',
	)
}
func TestKMapLookup6(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x41, 0,
		0x20, event.KSymSpace, ' ',
	)
}
func TestKMapLookup7(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x33, 0,
		0xfe53, event.KSymTilde, '~',
	)
}
func TestKMapLookup8(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x4d, 0,
		0xff7f, event.KSymNumLock, 'ｿ',
	)
}
func TestKMapLookup9(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x5b, 0,
		0xff9f, event.KSymKeypadDelete, 'ﾟ',
	)
}
func TestKMapLookup10(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x40, 0,
		0xffe9, event.KSymAltL, '￩',
	)
}
func TestKMapLookup11(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x74, km.mmask.shift|km.mmask.ctrl,
		0xff54, event.KSymDown, 'ｔ',
	)
}
func TestKMapLookup12(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x57, 0,
		0xff9c, event.KSymNone, 'ﾜ',
	)
	testLookup(t, km,
		0x57, km.mmask.numL,
		0xffb1, event.KSymKeypad1, '1',
	)
	testLookup(t, km,
		// shift not affecting keypad digit
		0x57, km.mmask.shift,
		0xff9c, event.KSymNone, 'ﾜ',
	)
	testLookup(t, km,
		// with numlock on, shift can affect keypad digit
		0x57, km.mmask.numL|km.mmask.shift,
		0xff9c, event.KSymNone, 'ﾜ',
	)
}
func TestKMapLookup13(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	kc := xproto.Keycode(0x5b) // keypad period/delete
	testLookup(t, km,
		kc, 0,
		0xff9f, event.KSymKeypadDelete, 'ﾟ',
	)
	testLookup(t, km,
		kc, km.mmask.numL,
		//0x2e, event.KSymPeriod, '.',
		0xffae, event.KSymKeypadDecimal, '.',
	)
}

//----------

func TestKMapLookup_FR1_1(t *testing.T) {
	km, conn := getKMap_FR1(t)
	defer conn.Close()

	kc := xproto.Keycode(0x5b) // keypad period/delete
	testLookup(t, km,
		kc, 0,
		0xff9f, event.KSymKeypadDelete, 'ﾟ',
	)
	testLookup(t, km,
		kc, km.mmask.numL,
		0x2e, event.KSymPeriod, '.',
	)
}

func TestKMapLookup_FR1_2(t *testing.T) {
	km, conn := getKMap_FR1(t)
	defer conn.Close()

	kc := xproto.Keycode(0x56) // keypad add
	testLookup(t, km,
		kc, km.mmask.numL,
		0xffab, event.KSymKeypadAdd, '+',
	)
	testLookup(t, km,
		kc, km.mmask.numL|km.mmask.shift,
		0x100002b, event.KSymNone, '+',
	)
}

func TestKMapLookup_FR1_3(t *testing.T) {
	km, conn := getKMap_FR1(t)
	defer conn.Close()

	kc := xproto.Keycode(0x3f) // keypad multiply
	testLookup(t, km,
		kc, 0,
		0xffaa, event.KSymKeypadMultiply, '*',
	)
	testLookup(t, km,
		kc, km.mmask.numL,
		0xffaa, event.KSymKeypadMultiply, '*',
	)
	testLookup(t, km,
		kc, km.mmask.numL|km.mmask.shift,
		0x10022c5, event.KSymNone, '⋅',
	)
	testLookup(t, km,
		kc, km.mmask.shift,
		0x10022c5, event.KSymNone, '⋅',
	)
}

// TODO: keypad divide "/" "∕"?

//----------

func TestDumpMapping(t *testing.T) {
	// comment this to enable
	t.Skip("avoid dumping in general tests")

	km, conn := getKMap(t)
	defer conn.Close()
	_ = km
	fmt.Print(km.Dump2())
}

//----------
//----------
//----------

func testLookup(
	t *testing.T,
	kmap *KMap,

	kc xproto.Keycode,
	kmods uint16,

	ks2 xproto.Keysym,
	eks2 event.KeySym,
	ru2 rune,
) {
	t.Helper()
	ks1, eks1, ru1 := kmap.Lookup(kc, kmods)
	if ks1 != ks2 || eks1 != eks2 || ru1 != ru2 {
		t.Fatalf("->(0x%x,%v)\nexp:(0x%x,%v,%q)\ngot:(0x%x,%v,%q)",
			kc, kmods,
			ks2, eks2, ru2,
			ks1, eks1, ru1,
		)
	}
}

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

	//fmt.Println(km.keysymsTableStr())

	return km, conn
}

//----------

func getKMap_FR1(t *testing.T) (*KMap, *xgb.Conn) {
	kmap, conn := getKMap(t)
	kmap.setKeyboardMappingEntries(keyboardMappingPartial_FR1())
	kmap.detectModifiersMapping(modifierMapping_FR1())
	return kmap, conn
}

func keyboardMappingPartial_FR1() map[xproto.Keycode][]xproto.Keysym {
	w := map[xproto.Keycode][]xproto.Keysym{
		// has KSymKeypadDelete, KSymPeriod
		0x5b: {0xff9f, 0x2e, 0xff9f, 0x2e, 0x2c, 0x100202f, 0x2c},
		// has KSymKeypadMultiply
		0x3f: {0xffaa, 0x10022c5, 0xffaa, 0x10022c5, 0x10000d7, 0xffffff, 0x1008fe21},
		// has KSymKeypadAdd
		0x56: {0xffab, 0x100002b, 0xffab, 0x100002b, 0x100002b, 0xffffff, 0x1008fe22},
	}
	return w
}

func modifierMapping_FR1() [8][]xproto.Keycode {
	return [8][]xproto.Keycode{
		{0x32, 0x3e, 0x0, 0x0},
		{0x42, 0x0, 0x0, 0x0},
		{0x25, 0x6d, 0x0, 0x0},
		{0x40, 0x9c, 0x0, 0x0},
		{0x4d, 0x0, 0x0, 0x0},
		{0x0, 0x0, 0x0, 0x0},
		{0x73, 0x74, 0x7f, 0x80},
		{0x8, 0x7c, 0x0, 0x0},
	}
}
