package xinput

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
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
		0xff7f, event.KSymNumLock, 0,
	)
}
func TestKMapLookup9(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x5b, 0,
		0xff9f, event.KSymKeypadDelete, 0,
	)
}
func TestKMapLookup10(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x40, 0,
		0xffe9, event.KSymAltL, 0,
	)
}
func TestKMapLookup11(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x74, km.mmask.shift|km.mmask.ctrl,
		0xff54, event.KSymDown, 0,
	)
}
func TestKMapLookup12(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	testLookup(t, km,
		0x57, 0,
		0xff9c, event.KSymNone, 0,
	)
	testLookup(t, km,
		0x57, km.mmask.numL,
		0xffb1, event.KSymKeypad1, '1',
	)
	testLookup(t, km,
		// shift not affecting keypad digit
		0x57, km.mmask.shift,
		0xff9c, event.KSymNone, 0,
	)
	testLookup(t, km,
		// with numlock on, shift can affect keypad digit
		0x57, km.mmask.numL|km.mmask.shift,
		0xff9c, event.KSymNone, 0,
	)
}
func TestKMapLookup13(t *testing.T) {
	km, conn := getKMap(t)
	defer conn.Close()

	kc := xproto.Keycode(0x5b) // keypad period/delete
	testLookup(t, km,
		kc, 0,
		0xff9f, event.KSymKeypadDelete, 0,
	)
	testLookup(t, km,
		kc, km.mmask.numL,
		//0x2e, event.KSymPeriod, '.',
		0xffae, event.KSymKeypadDecimal, '.',
	)
}

//----------

func TestKMapLookup_FR1_1(t *testing.T) {
	km := getKMap_FR1(t)

	kc := xproto.Keycode(0x5b) // keypad period/delete
	testLookup(t, km,
		kc, 0,
		0xff9f, event.KSymKeypadDelete, 0,
	)
	testLookup(t, km,
		kc, km.mmask.numL,
		0x2e, event.KSymPeriod, '.',
	)
}

func TestKMapLookup_FR1_2(t *testing.T) {
	km := getKMap_FR1(t)

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
	km := getKMap_FR1(t)

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

var testXvfb = flag.Bool("xinput-xvfb", false, "run keyboard integration tests using a private Xvfb server")

// Issues #82/#83: run without an X server using go test ./driver/xdriver/xinput -run '^Test(KMapUSCZReproduction|KMapLookup_FR1|KeysymUnicode|XKB)' -count=1 -v.
func TestKMapUSCZReproduction(t *testing.T) {
	km := &KMap{}
	// Core symbols captured from Linux Xvfb; the reduced XKB fixture below supplies group and level rules.
	km.setKeyboardMappingEntries(map[xproto.Keycode][]xproto.Keysym{
		11: {0x32, 0x40, 0x1ec, 0x32, 0x40, 0xfe5a},
		22: {0xff08, 0xff08, 0xff08, 0xff08},
		26: {0x65, 0x45, 0x65, 0x45, 0x20ac, 0x45},
		36: {0xff0d, 0, 0xff0d},
		38: {0x61, 0x41, 0x61, 0x41, 0x7e, 0xc6},
		65: {0x20, 0xfe08, 0x20, 0xfe08},
	})
	km.detectModifiersMapping([8][]xproto.Keycode{})
	km.xkb = usczXKBFixture()
	for _, tc := range usczCases() {
		t.Run(tc.name, func(t *testing.T) {
			ks, _, ru := km.Lookup(tc.keycode, tc.state)
			checkUSCZLookup(t, tc, ks, ru)
		})
	}
}

// Run with go test ./driver/xdriver/xinput -run '^TestKMapUSCZIntegration$' -count=1 -v -timeout 30s -xinput-xvfb.
// Requires Xvfb, setxkbmap and XKB data; injection uses xgb directly without CGo or helper scripts.
// The test owns and terminates a private X server and never changes the desktop keyboard configuration.
// Group selection is direct, so this does not test the Super+Space shortcut or its unwanted space insertion.
func TestKMapUSCZIntegration(t *testing.T) {
	if !*testXvfb {
		t.Skip("use -xinput-xvfb to run against a private Xvfb server")
	}
	for _, tool := range []string{"Xvfb", "setxkbmap"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Authentication is disabled only for this private local server; TCP connections are disabled.
	server := exec.CommandContext(ctx, "Xvfb", "-displayfd", "1", "-screen", "0", "800x600x24", "-nolisten", "tcp", "-noreset", "-ac")
	stdout, err := server.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	server.Stderr = os.Stderr
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	})
	number, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	display := ":" + strings.TrimSpace(number)
	setup := exec.CommandContext(ctx, "setxkbmap", "-display", display, "-model", "pc105", "-layout", "us,cz", "-option", "", "-option", "grp:win_space_toggle")
	if out, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("setxkbmap: %v: %s", err, out)
	}
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	injector, err := newUSCZInjector(display)
	if err != nil {
		t.Fatal(err)
	}
	defer injector.conn.Close()
	km, err := NewKMap(conn)
	if err != nil {
		t.Fatal(err)
	}
	screen := xproto.Setup(conn).DefaultScreen(conn)
	window, err := xproto.NewWindowId(conn)
	if err != nil {
		t.Fatal(err)
	}
	if err := xproto.CreateWindowChecked(conn, screen.RootDepth, window, screen.Root, 0, 0, 200, 100, 0, xproto.WindowClassInputOutput, screen.RootVisual, xproto.CwEventMask, []uint32{xproto.EventMaskKeyPress}).Check(); err != nil {
		t.Fatal(err)
	}
	if err := xproto.MapWindowChecked(conn, window).Check(); err != nil {
		t.Fatal(err)
	}
	if err := xproto.SetInputFocusChecked(conn, xproto.InputFocusPointerRoot, window, xproto.TimeCurrentTime).Check(); err != nil {
		t.Fatal(err)
	}
	t.Logf("display=%s AltGr mask=%#x", display, km.mmask.altGr)
	for _, kc := range []xproto.Keycode{11, 22, 26, 36, 38, 65} {
		t.Logf("keycode=%d symbols=%#x", kc, km.kbm[kc])
	}
	// Closing the connection also releases WaitForEvent if the server stops replying.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()
	runCase := func(tc usczCase) {
		t.Run(tc.name, func(t *testing.T) {
			if err := injector.inject(tc); err != nil {
				t.Fatalf("inject: %v", err)
			}
			for {
				ev, err := conn.WaitForEvent()
				if err != nil || ev == nil {
					t.Fatalf("waiting for key: %v", err)
				}
				switch e := ev.(type) {
				case xproto.MappingNotifyEvent:
					t.Logf("MappingNotify request=%d", e.Request)
					if err := km.ReadMapping(); err != nil {
						t.Fatal(err)
					}
				case xproto.KeyPressEvent:
					if e.Detail != tc.keycode {
						continue
					}
					ks, _, ru := km.Lookup(e.Detail, e.State)
					t.Logf("group=%d keycode=%d state=%#x", tc.group, e.Detail, e.State)
					// XKB-aware clients receive the group in bits 13-14, not the old compatibility AltGr modifier.
					if group := int(e.State>>13) & 3; group != tc.group {
						t.Errorf("event group=%d, want %d", group, tc.group)
					}
					checkUSCZLookup(t, tc, ks, ru)
					return
				}
			}
		})
	}
	for _, tc := range usczCases() {
		runCase(tc)
	}
	// Replacing the map must reload XKB types and symbols through the normal MappingNotify path.
	setup = exec.CommandContext(ctx, "setxkbmap", "-display", display, "-model", "pc105", "-layout", "cz,us", "-option", "", "-option", "grp:win_space_toggle")
	if out, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("replace map: %v: %s", err, out)
	}
	runCase(usczCase{name: "ChangedMap/CZ2", keycode: 11, keysym: 0x1ec, text: '\u011b'})
}

//----------

func TestXKBDecodeMapping(t *testing.T) {
	// One two-level type with a preserve entry and one key with two symbols.
	reply := make([]byte, 40+8+8+4+8+8)
	xgb.Put16(reply[12:], 3)
	reply[15], reply[17], reply[20] = 1, 38, 1
	reply[40], reply[44], reply[45], reply[46] = 1, 2, 1, 1
	reply[48], reply[49], reply[50] = 1, 1, 1
	reply[64], reply[65] = 1, 2
	xgb.Put16(reply[66:], 2)
	xgb.Put32(reply[68:], 0x61)
	xgb.Put32(reply[72:], 0x41)
	m := &xkbMap{}
	if err := m.decodeMapping(reply); err != nil {
		t.Fatal(err)
	}
	if got := m.lookup(38, 0); got != 0x61 {
		t.Fatalf("plain: %#x", got)
	}
	if got := m.lookup(38, 1); got != 0x41 {
		t.Fatalf("shift: %#x", got)
	}
	for i := range len(reply) {
		if err := m.decodeMapping(reply[:i]); !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("truncated at %d: %v", i, err)
		}
		if got := m.lookup(38, 1); got != 0x41 {
			t.Fatalf("failed decoding changed the map: %#x", got)
		}
	}
}

func TestXKBGroupBounds(t *testing.T) {
	for _, tc := range []struct {
		info byte
		want uint32
	}{
		{2, 0x61},    // Wrap group 3 into group 1.
		{0x42, 0x62}, // Clamp to group 2.
		{0x92, 0x62}, // Redirect to group 2.
		{0xb2, 0x61}, // Invalid redirect falls back to group 1.
	} {
		m := usczXKBFixture()
		k := &m.keys[38]
		k.groupInfo = tc.info
		k.symbols[2] = 0x62
		if got := uint32(m.lookup(38, 2<<13)); got != tc.want {
			t.Errorf("group info %#x: got %#x, want %#x", tc.info, got, tc.want)
		}
	}
}

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

func getKMap_FR1(t *testing.T) *KMap {
	t.Helper()
	kmap := &KMap{}
	kmap.setKeyboardMappingEntries(keyboardMappingPartial_FR1())
	kmap.detectModifiersMapping(modifierMapping_FR1())
	return kmap
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

//----------

// Use a separate connection for group selection and physical key injection.
type usczInjector struct {
	conn   *xgb.Conn
	opcode byte
}

func newUSCZInjector(display string) (*usczInjector, error) {
	conn, err := xgb.NewConnDisplay(display)
	if err != nil {
		return nil, err
	}
	inj := &usczInjector{conn: conn}
	if err := inj.init(); err != nil {
		conn.Close()
		return nil, err
	}
	return inj, nil
}

func (inj *usczInjector) init() error {
	if err := xtest.Init(inj.conn); err != nil {
		return err
	}
	const name = "XKEYBOARD"
	ext, err := xproto.QueryExtension(inj.conn, uint16(len(name)), name).Reply()
	if err != nil {
		return err
	}
	if !ext.Present {
		return fmt.Errorf("XKEYBOARD extension is unavailable")
	}
	inj.opcode = ext.MajorOpcode
	// XKB UseExtension (minor opcode 0), requesting protocol version 1.0; see XKBproto.h.
	buf := make([]byte, 8)
	buf[0] = inj.opcode
	xgb.Put16(buf[2:], 2)
	xgb.Put16(buf[4:], 1)
	cookie := inj.conn.NewCookie(true, true)
	inj.conn.NewRequest(buf, cookie)
	reply, err := cookie.Reply()
	if err != nil {
		return err
	}
	if len(reply) < 32 || reply[1] == 0 {
		return fmt.Errorf("XKB protocol version 1.0 is unsupported")
	}
	return nil
}

func (inj *usczInjector) inject(tc usczCase) error {
	// XKB LatchLockState (minor opcode 5), changing only the group lock on UseCoreKbd (0x100).
	buf := make([]byte, 16)
	buf[0], buf[1] = inj.opcode, 5
	xgb.Put16(buf[2:], 4)
	xgb.Put16(buf[4:], 0x100)
	buf[6], buf[7] = 2, byte(tc.state&2) // Reset or set CapsLock for each case.
	buf[8], buf[9] = 1, byte(tc.group)
	cookie := inj.conn.NewCookie(true, false)
	inj.conn.NewRequest(buf, cookie)
	if err := cookie.Check(); err != nil {
		return err
	}
	key := func(kind byte, code xproto.Keycode) error {
		return xtest.FakeInputChecked(inj.conn, kind, byte(code), 0, 0, 0, 0, 0).Check()
	}
	if tc.state&1 != 0 {
		shift := tc.shiftKey
		if shift == 0 {
			shift = 50
		}
		if err := key(xproto.KeyPress, shift); err != nil {
			return err
		}
		defer key(xproto.KeyRelease, shift)
	}
	if tc.state&0x80 != 0 {
		if err := key(xproto.KeyPress, 108); err != nil {
			return err
		}
		defer key(xproto.KeyRelease, 108)
	}
	if err := key(xproto.KeyPress, tc.keycode); err != nil {
		return err
	}
	return key(xproto.KeyRelease, tc.keycode)
}

//----------

type usczCase struct {
	name     string
	group    int
	keycode  xproto.Keycode
	state    uint16
	keysym   xproto.Keysym
	text     rune
	shiftKey xproto.Keycode
}

func usczCases() []usczCase {
	// Before the fix, Linux reproduced Shift+Space -> U+FE08, CZ 2 -> @ and CZ Enter/Backspace -> NoSymbol.
	// Linux produced CZ a -> ~ and e -> euro; the OpenBSD reporter instead observed no response for a and ~ for e.
	return []usczCase{
		{"US/a", 0, 38, 0, 0x61, 'a', 0},
		{"US/e", 0, 26, 0, 0x65, 'e', 0},
		{"US/2", 0, 11, 0, 0x32, '2', 0},
		{"US/ShiftSpace", 0, 65, 1, 0x20, ' ', 50},
		{"US/RightShiftSpace", 0, 65, 1, 0x20, ' ', 62},
		{"US/ShiftA", 0, 38, 1, 0x41, 'A', 0},
		{"US/CapsA", 0, 38, 2, 0x41, 'A', 0},
		{"US/CapsShiftA", 0, 38, 3, 0x61, 'a', 0},
		{"CZ/a", 1, 38, 0x2000, 0x61, 'a', 0},
		{"CZ/e", 1, 26, 0x2000, 0x65, 'e', 0},
		{"CZ/2", 1, 11, 0x2000, 0x1ec, '\u011b', 0},
		{"CZ/Shift2", 1, 11, 0x2001, 0x32, '2', 0},
		{"CZ/AltGrE", 1, 26, 0x2080, 0x20ac, '\u20ac', 0},
		{"CZ/Enter", 1, 36, 0x2000, 0xff0d, 0, 0},
		{"CZ/Backspace", 1, 22, 0x2000, 0xff08, 0, 0},
		{"USAgain/Enter", 0, 36, 0, 0xff0d, 0, 0},
		{"USAgain/Backspace", 0, 22, 0, 0xff08, 0, 0},
	}
}

// Reduced Linux US/CZ model for the tested groups and levels, not a complete map captured on OpenBSD.
func usczXKBFixture() *xkbMap {
	m := &xkbMap{types: []xkbKeyType{
		{},
		{mask: 1, entries: []xkbTypeEntry{{mask: 1, level: 1}}},
		{mask: 0x40, entries: []xkbTypeEntry{{mask: 0x40, level: 1}}},
		{mask: 3, entries: []xkbTypeEntry{{mask: 1, level: 1}, {mask: 2, level: 1}}},
		{mask: 0x81, entries: []xkbTypeEntry{{mask: 1, level: 1}, {mask: 0x80, level: 2}, {mask: 0x81, level: 3}}},
	}}
	m.keys[11] = xkbKey{types: [4]byte{1, 1}, groupInfo: 2, width: 2, symbols: []xproto.Keysym{0x32, 0x40, 0x1ec, 0x32}}
	m.keys[26] = xkbKey{types: [4]byte{3, 4}, groupInfo: 2, width: 4, symbols: []xproto.Keysym{0x65, 0x45, 0, 0, 0x65, 0x45, 0x20ac, 0x45}}
	m.keys[38] = xkbKey{types: [4]byte{3, 3}, groupInfo: 2, width: 2, symbols: []xproto.Keysym{0x61, 0x41, 0x61, 0x41}}
	m.keys[22] = xkbKey{groupInfo: 1, width: 1, symbols: []xproto.Keysym{0xff08}}
	m.keys[36] = xkbKey{groupInfo: 1, width: 1, symbols: []xproto.Keysym{0xff0d}}
	m.keys[65] = xkbKey{types: [4]byte{2}, groupInfo: 1, width: 2, symbols: []xproto.Keysym{0x20, 0xfe08}}
	return m
}

func checkUSCZLookup(t *testing.T, tc usczCase, ks xproto.Keysym, ru rune) {
	t.Helper()
	// Control keys are dispatched by keysym; text == 0 intentionally leaves their rune unspecified.
	if ks != tc.keysym || tc.text != 0 && ru != tc.text {
		t.Errorf("got keysym=%#x rune=%U; want keysym=%#x text=%U", ks, ru, tc.keysym, tc.text)
	}
}
