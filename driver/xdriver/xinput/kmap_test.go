package xinput

import (
	"os"
	"testing"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/jmigpin/editor/util/uiutil/event"
)

func TestKMapLookup1(t *testing.T) {
	kmap, conn, err := getKMap(t)
	if err != nil {
		t.Fatal(err)
	}
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
	}

	//println(kmap.KeysymTable())

	//for i := kmap.si.MinKeycode; i < 255; i++ {
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

func getKMap(t *testing.T) (*KMap, *xgb.Conn, error) {
	display := os.Getenv("DISPLAY")
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		return nil, nil, err
	}
	km, err := NewKMap(conn)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return km, conn, err
}
