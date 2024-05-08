package debug

import (
	"regexp"
	"testing"
	"unsafe"
)

func TestStringify(t *testing.T) {
	a := int(1)
	runTest1(t, a, "1")

	b := int(1)
	//runTest1(t, &b, "@0x.*")
	runTest1(t, &b, "&1")

	var c *int
	runTest1(t, c, "nil")
	runTest1(t, &c, "&nil")
}

func TestStringifyPtr2(t *testing.T) {
	type St1 struct {
		a int
		B int
	}
	var a St1
	runTest1(t, &a, "&St1{0 0}")
	//runTest1(t, &a, "&{0 0}")
}

func TestStringifyPtr3(t *testing.T) {
	type St1 struct {
		a int
		B int
	}
	var v1 *St1
	runTest1(t, v1, "nil")

	v2 := &St1{2, 3}
	runTest1(t, v2, "&St1{2 3}")
	//runTest1(t, v2, "&{2 3}")

	v3 := &v2
	//runTest1(t, &v3, "@0x.*")
	runTest1(t, &v3, "&&&St1{2 3}")
	//runTest1(t, &v3, "&&&{2 3}")
}

func TestStringifyUintptr(t *testing.T) {
	type Handle uintptr
	a := Handle(0)
	runTest1(t, a, "0x0")
	b := Handle(1)
	runTest1(t, b, "0x1")
}

func TestStringifyMap(t *testing.T) {
	//a := map[string]int{"a": 1, "b": 2}
	//runTest1(t, a, "map[\"a\":1 \"b\":2]") // TODO: key order
}

func TestStringifySlice(t *testing.T) {
	type S1 struct {
		a int
		B int
	}
	a := []*S1{{1, 2}, {3, 4}, {5, 6}}
	runTest1(t, a, "[&{1 2} &{3 4} &{5 6}]")
	runTest1(t, a[1:1], "[]")
	runTest1(t, a[1:2], "[&{3 4}]")

	b := []*S1{nil, nil}
	runTest1(t, b, "[nil nil]")
}

func TestStringifySlice2(t *testing.T) {
	type S1 struct {
		a int
		B int
		c any
	}
	a := []*S1{{1, 2, 10}, {3, 4, true}}
	runTest1(t, a, "[&{1 2 10} &{3 4 true}]")

	type S2 struct{ b bool }
	b := []*S1{{1, 2, S2{true}}, {3, 4, &S2{false}}}
	//runTest1(t, b, "@\\[&{1 2 S2{true}} &{3 4 0x.*]")
	//runTest2(t, b, "@\\[&{1 2 S2{true}} &{3 4 0x.*]", 150, 5)
	runTest2(t, b, "@\\[&{1 2 S2{true}} &{3 4 &S2{...}}]", 100, 5)
	runTest2(t, b, "[&{1 2 S2{true}} &{3 4 &S2{false}}]", 50, 10)

	c := []*S1{{c: &S1{c: &S2{true}}}}
	//runTest1(t, c, "@\\[&{0 0 0x.*}]")
	runTest2(t, c, "[&{0 0 &S1{...}}]", 100, 4)
	runTest2(t, c, "[&{0 0 &S1{0 0 &S2{true}}}]", 50, 10)
}

func TestStringifyArray(t *testing.T) {
	type S1 struct {
		a int
		B int
	}
	a := [...]*S1{{1, 2}, {3, 4}, {5, 6}}
	runTest1(t, a, "[&{1 2} &{3 4} &{5 6}]")
}

func TestStringifyInterface(t *testing.T) {
	type S1 struct {
		a int
		B int
		c any
	}
	var a any = &S1{1, 2, 3}
	runTest1(t, a, "&S1{1 2 3}")
	runTest1(t, &a, "&&S1{1 2 3}")
	var c = &S1{1, 2, 3}
	runTest1(t, c, "&S1{1 2 3}")

	var b any = &a
	runTest1(t, b, "&&S1{1 2 3}")
}

func TestStringifyChan(t *testing.T) {
	a := make(chan int)
	runTest1(t, a, "@0x.*")
	var b chan int
	runTest1(t, &b, "&0x0")
	var c *chan int
	runTest1(t, c, "nil")
	var d *chan int
	runTest1(t, &d, "&nil")
}

func TestStringifyUnsafePointer(t *testing.T) {
	a := 5
	b := unsafe.Pointer(&a)
	runTest1(t, b, "@0x.*")
}

func TestStringifyBytes(t *testing.T) {
	a := []byte("abc")
	runTest1(t, a, "[97 98 99]")

	a2 := []byte{}
	runTest1(t, a2, "[]")

	b := []byte{1, 2, 3, 'a'}
	runTest1(t, b, "[1 2 3 97]")
	//runTest2(t, b, "[1 2 ...]", 2, 3)
	runTest2(t, b, "[1 2 ...]", 4, 3)

	type S1 struct {
		a []byte
	}
	c := &S1{[]byte{1, 2, 3}}
	runTest1(t, c, "&S1{[1 2 3]}")

	//d := []byte{1, 2, 3}
	//println(d)
	//fmt.Printf("%v\n", d)
	//fmt.Printf("%s\n", d)
}

func TestStringifyBytes2(t *testing.T) {
	b := []byte{'a', 'b', 'c'}
	for i := 0; i < 10; i++ {
		b = append(b, b...)
	}
	runTest3(t, b, "\"abcabcabc...\"", 10, 5, true)
}

func TestStringifyBytes3(t *testing.T) {
	type t1 struct {
		b []byte
	}
	v1 := &t1{}
	runTest3(t, v1, "&t1{\"\"}", 100, 55, true)

	v2 := &t1{b: []byte("abc")}
	runTest3(t, v2, "&t1{\"abc\"}", 100, 55, true)
}

func TestStringifyBytes4(t *testing.T) {
	type t3 []byte
	type t2 struct {
		t3 t3
	}
	type t1 struct {
		b  []byte
		t2 t2
	}
	v1 := &t1{}
	runTest3(t, v1, "&t1{\"\" {\"\"}}", 100, 55, true)
}

func TestStringifyString(t *testing.T) {
	b := []byte{'a', 'b', 'c'}
	for i := 0; i < 10; i++ {
		b = append(b, b...)
	}
	c := string(b)
	runTest2(t, c, "\"abcabcabc...\"", 10, 5)
}

func TestStringifyRunes(t *testing.T) {
	b := []byte{'a', 'b', 'c'}
	for i := 0; i < 10; i++ {
		b = append(b, b...)
	}
	c := []rune(string(b))
	runTest3(t, c, "\"abcabcabc...\"", 10, 5, true)
}

func TestStringifyNilReceiver(t *testing.T) {
	var p *Dummy1
	runTest1(t, p, "nil")

	a := uintptr(0)
	var b *Dummy1 = (*Dummy1)(unsafe.Pointer(a))
	runTest1(t, b, "nil")
}

func TestStringifyPanic(t *testing.T) {
	// DISABLED: keeps failing to run recover() (?)
	t.Logf("disabled")

	//a := uintptr(1)
	//b := (*Dummy1)(unsafe.Pointer(a))
	//runTest1(t, b, "&Dummy1{\"\"PANIC}") // NOTE: string cut short due to panic(?)
}

func TestStringifyStringError(t *testing.T) {
	v1 := &Dummy1{"aa"}
	runTest1(t, v1, "&Dummy1{\"aa\"}")

	v2 := &Dummy2{"bb"}
	runTest1(t, v2, "&Dummy2{\"bb\"}")
}

func TestStringifyNil(t *testing.T) {
	a := any(nil)
	runTest1(t, a, "nil")
	runTest1(t, &a, "&nil")
}

//----------

//func TestSliceCut(t *testing.T) {
//	b1 := []interface{}{}
//	for i := 0; i < 50; i++ {
//		b1 = append(b1, i)
//	}
//	t.Logf("%0.5v\n", b1) // not trimmed
//	t.Logf("%0.5q\n", b1)
//	t.Logf("%0.5x\n", b1)
//	t.Logf("%0.5s\n", b1)
//	t.Logf("%0.2v\n", true)
//}

//----------

func runTest1(t *testing.T, v any, out string) {
	t.Helper()
	runTest2(t, v, out, 0, 0)
}
func runTest2(t *testing.T, v any, out string, max, maxDepth int) {
	t.Helper()
	runTest3(t, v, out, max, maxDepth, false)
}
func runTest3(t *testing.T, v any, out string, max, maxDepth int, sbr bool) {
	t.Helper()
	s2 := ""
	if max == 0 && maxDepth == 0 {
		// use production values
		s2 = stringifyV3(v)
	} else {
		p := newPrint3(max, maxDepth, sbr)
		p.do(v)
		s2 = p.ToString()
	}

	// support regular expression match if starting with @
	res := false
	if len(out) > 0 && out[0] == '@' {
		out = out[1:]
		m, err := regexp.MatchString("^"+out+"$", s2)
		if err != nil {
			panic(err)
		}
		res = m
	} else {
		res = s2 == out
	}

	if !res {
		t.Fatalf("got %q expecting %q", s2, out)
	}
}

//----------

type Dummy1 struct{ s string }

func (d *Dummy1) String() string { return d.s }

type Dummy2 struct{ s string }

func (d *Dummy2) Error() string { return d.s }
