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
		eks   event.KeySym
		ru    rune
	}
	pairs := []pair{
		{11, 0, event.KSym2, '2'},
		{11, xproto.KeyButMaskMod5, event.KSymAt, '@'},
		{38, 0, event.KSymA, 'a'},
		{38, xproto.KeyButMaskShift, event.KSymA, 'A'},
		{35, xproto.KeyButMaskShift, event.KSymGrave, '`'},
		{65, 0, event.KSymSpace, ' '},
		{51, 0, event.KSymTilde, '~'},
		{77, 0, event.KSymNumLock, 'ｿ'},
		{91, 0, event.KSymKeypadDelete, 'ﾟ'},
		{91, xproto.KeyButMaskMod2, event.KSymKeypadDecimal, '.'},
	}

	//println(kmap.KeysymTable())

	//for i := 0; i < 256; i++ {
	//	eks, ru := kmap.Lookup(xproto.Keycode(i), 0)
	//	fmt.Printf("%v, %v, %v(%c)\n", i, eks, ru, ru)
	//}

	for _, p := range pairs {
		eks, ru := kmap.Lookup(p.kc, p.kmods)
		if eks != p.eks || ru != p.ru {
			t.Logf("%v, %v, %v(%c)\n", p.kc, eks, ru, ru)
			t.Fail()
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
