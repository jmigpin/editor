package core

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestReadLastUntilStart(t *testing.T) {
	type in struct {
		str   string
		index int
	}
	type out struct {
		start int
		str   string
		ok    bool
	}
	type result struct {
		in  in
		out out
	}
	w := []result{
		{in{"aa)bbb", 6}, out{3, "bbb", true}},
		{in{"abc", 3}, out{0, "abc", true}},
		{in{"", 0}, out{0, "", false}},
		{in{"  ", 2}, out{0, "", false}},
		{in{".", 1}, out{0, "", false}},
	}
	for _, u := range w {
		rw := iorw.NewStringReaderAt(u.in.str)
		start, str, ok := readLastUntilStart(rw, u.in.index)
		if !(start == u.out.start && str == u.out.str && ok == u.out.ok) {
			t.Fatal(start, str, ok, "expecting", u.out)
		}
	}
}

func TestExpandAndFilter(t *testing.T) {
	type in struct {
		completions []string
		prefix      string // text already written
	}
	type out struct {
		expand string
		comps  []string
	}
	type result struct {
		in  in
		out out
	}
	w := []result{
		// basic completion
		{
			in{[]string{"Print", "Println"}, "pr"},
			out{"Print", []string{"Print", "Println"}},
		},
		// unique completion
		{
			in{[]string{"Println"}, "Print"},
			out{"Println", []string{"Println"}},
		},
		// iterate completion casing
		{
			in{[]string{"PrintA", "PrintA2", "Printa3"}, "printa"},
			out{"PrintA", []string{"PrintA", "PrintA2", "Printa3"}},
		},
		{
			in{[]string{"PrintA", "PrintA2", "Printa3"}, "PrintA"},
			out{"Printa", []string{"PrintA", "PrintA2", "Printa3"}},
		},
		{
			in{[]string{"UaAa", "UaAA"}, "UaAA"},
			out{"UaAa", []string{"UaAa", "UaAA"}},
		},
		// other tests
		{
			in{[]string{"aaa", "aaabbb"}, "aa"},
			out{"aaa", []string{"aaa", "aaabbb"}},
		},
		{
			in{[]string{"aaa", "aAa"}, "aa"},
			out{"aaa", []string{"aaa", "aAa"}},
		},
		{
			in{[]string{"aaa", "aAa"}, "aaa"},
			out{"aAa", []string{"aaa", "aAa"}},
		},
		{
			in{[]string{"aAa"}, "aaa"},
			out{"aAa", []string{"aAa"}},
		},
		{
			in{[]string{"abCCCe"}, "abC"},
			out{"abCCCe", []string{"abCCCe"}},
		},
		{
			in{[]string{"read", "receive", "Recv"}, "r"},
			out{"re", []string{"read", "receive", "Recv"}},
		},
		{
			in{[]string{"Recv", "read", "receive"}, "R"},
			out{"Re", []string{"Recv", "read", "receive"}},
		},
		{
			in{[]string{"u", "abcd", "abc"}, "a"},
			out{"abc", []string{"abcd", "abc"}},
		},
		{
			in{[]string{"lockaaa", "Lock", "lockaaab"}, "lock"},
			out{"Lock", []string{"lockaaa", "Lock", "lockaaab"}},
		},
		{
			in{[]string{"builder", "Build"}, "buil"},
			out{"build", []string{"builder", "Build"}},
		},
		{
			in{[]string{"builder", "Build"}, "build"},
			out{"Build", []string{"builder", "Build"}},
		},
	}
	for _, u := range w {
		expand, comps := expandAndFilter(u.in.prefix, u.in.completions)
		if !(expand == u.out.expand && cmpStrSlices(comps, u.out.comps)) {
			t.Fatal("expecting:\n", u.out, "\ngot:\n", expand, comps)
		}
	}
}

func TestInsertComplete(t *testing.T) {
	type in struct {
		comps []string
		text  string
		index int
	}
	type out struct {
		newIndex  int
		completed bool
		comps     []string
		text      string
	}
	type result struct {
		in  in
		out out
	}
	w := []result{
		{
			in{[]string{"aaa", "aaabbb"}, "aa", 2},
			out{3, true, []string{"aaa", "aaabbb"}, "aaa"},
		},
		{
			in{[]string{"aAa", "aaa"}, "aa", 2},
			out{3, true, []string{"aAa", "aaa"}, "aAa"},
		},
		{
			in{[]string{"aAac"}, "aaac", 4},
			out{4, true, []string{"aAac"}, "aAac"},
		},
		{
			in{[]string{"aa"}, "aa", 2},
			out{0, false, []string{"aa"}, "aa"},
		},
		{
			in{[]string{"abc"}, "abc", 1},
			out{3, true, []string{"abc"}, "abc"},
		},
		{
			in{[]string{"u", "abcd", "abc"}, "a", 1},
			out{3, true, []string{"abcd", "abc"}, "abc"},
		},
		{
			in{[]string{"abcd", "abc"}, "abe", 1},
			out{3, true, []string{"abcd", "abc"}, "abce"},
		},
		{
			in{[]string{"aaBbbCcc"}, "aaCc", 4},
			out{0, false, nil, "aaCc"},
		},
		{
			in{[]string{"Recv", "read", "receive"}, "Recv", 1},
			out{2, true, []string{"Recv", "read", "receive"}, "Recv"},
		},
	}
	for _, u := range w {
		rw := iorw.NewBytesReadWriterAt([]byte(u.in.text))
		newIndex, completed, comps, _ := insertComplete(u.in.comps, rw, u.in.index)
		b, err := iorw.ReadFastFull(rw)
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if !(cmpStrSlices(comps, u.out.comps) &&
			text == u.out.text &&
			newIndex == u.out.newIndex &&
			completed == u.out.completed) {
			//t.Fatal(newIndex, completed, comps, text, "expecting", u.out)
			t.Fatal("expecting:\n", u.out, "\ngot:\n", newIndex, completed, comps, text)
		}
	}
}

//----------

func cmpStrSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, s := range a {
		if s != b[i] {
			return false
		}
	}
	return true
}
