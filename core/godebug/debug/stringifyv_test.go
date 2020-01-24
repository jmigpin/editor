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

	v3 := &v2
	//runTest1(t, &v3, "@0x.*")
	runTest1(t, &v3, "&&&St1{2 3}")
}

func TestStringifyUintptr(t *testing.T) {
	type Handle uintptr
	var u Handle
	runTest1(t, u, "nil")
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
	a := []*S1{&S1{1, 2}, &S1{3, 4}, &S1{5, 6}}
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
		c interface{}
	}
	a := []*S1{&S1{1, 2, 10}, &S1{3, 4, true}}
	runTest1(t, a, "[&{1 2 10} &{3 4 true}]")

	type S2 struct{ b bool }
	b := []*S1{&S1{1, 2, S2{true}}, &S1{3, 4, &S2{false}}}
	runTest1(t, b, "@\\[&{1 2 S2{true}} &{3 4 0x.*]")
	runTest2(t, b, "[&{1 2 S2{true}} &{3 4 &S2{false}}]", 50, 10)

	c := []*S1{&S1{c: &S1{c: &S2{true}}}}
	runTest1(t, c, "@\\[&{0 0 0x.*}]")
	runTest2(t, c, "[&{0 0 &S1{0 0 &S2{true}}}]", 50, 10)
}

func TestStringifyArray(t *testing.T) {
	type S1 struct {
		a int
		B int
	}
	a := [...]*S1{&S1{1, 2}, &S1{3, 4}, &S1{5, 6}}
	runTest1(t, a, "[&{1 2} &{3 4} &{5 6}]")
}

func TestStringifyInterface(t *testing.T) {
	type S1 struct {
		a int
		B int
		c interface{}
	}
	var a interface{} = &S1{1, 2, 3}
	runTest1(t, a, "&S1{1 2 3}")

	var b interface{} = &a
	runTest1(t, b, "&&S1{1 2 3}")
}

func TestStringifyChan(t *testing.T) {
	a := make(chan int)
	runTest1(t, a, "@0x.*")
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
	runTest2(t, b, "[1 2 ...]", 2, 3)

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

//----------

func TestSliceCut(t *testing.T) {
	b1 := []interface{}{}
	for i := 0; i < 50; i++ {
		b1 = append(b1, i)
	}
	t.Logf("%0.5v\n", b1) // not trimmed
	t.Logf("%0.5q\n", b1)
	t.Logf("%0.5x\n", b1)
	t.Logf("%0.5s\n", b1)
	t.Logf("%0.2v\n", true)
}

//----------

func runTest1(t *testing.T, v interface{}, out string) {
	t.Helper()
	runTest2(t, v, out, 0, 0)
}
func runTest2(t *testing.T, v interface{}, out string, max, maxPtrDepth int) {
	t.Helper()

	s2 := ""
	if max == 0 && maxPtrDepth == 0 {
		s2 = stringifyV2(v) // use production values
	} else {
		p := NewPrint(max, maxPtrDepth)
		s2 = string(p.Do(v))
	}

	// support regular expression match if starting with @
	res := false
	if out[0] == '@' {
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
		s := stringifyV1(v)
		t.Fatalf("got %q expecting %q (alt: %v)", s2, out, s)
	}
}
