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
		rw := iorw.NewStringReader(u.in.str)
		start, str, ok := readLastUntilStart(rw, u.in.index)
		if !(start == u.out.start && str == u.out.str && ok == u.out.ok) {
			t.Fatal(start, str, ok, "expecting", u.out)
		}
	}
}

func TestFilterPrefixedAndExpand(t *testing.T) {
	type in struct {
		comps  []string
		prefix string
	}
	type out struct {
		expand      int
		canComplete bool
		comps       []string
	}
	type result struct {
		in  in
		out out
	}
	w := []result{
		{
			in{[]string{"aaa", "aaabbb"}, "aa"},
			out{1, true, []string{"aaa", "aaabbb"}},
		},
		{
			in{[]string{"aaa", "aAa"}, "aa"},
			out{0, false, []string{"aaa", "aAa"}},
		},
		{
			in{[]string{"aaa", "aAa"}, "aaa"},
			out{0, false, []string{"aaa", "aAa"}},
		},
		{
			in{[]string{"aAa"}, "aaa"},
			out{0, true, []string{"aAa"}},
		},
		{
			in{[]string{"aaabbbCCCe"}, "aaabbbC"},
			out{3, true, []string{"aaabbbCCCe"}},
		},
	}
	for _, u := range w {
		expand, canComplete, comps := filterPrefixedAndExpand(u.in.comps, u.in.prefix)
		if !(expand == u.out.expand &&
			canComplete == u.out.canComplete &&
			cmpStrSlices(comps, u.out.comps)) {
			t.Fatal(expand, canComplete, comps, "expecting", u.out)
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
			out{0, false, []string{"aAa", "aaa"}, "aa"},
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
	}
	for _, u := range w {
		rw := iorw.NewBytesReadWriter([]byte(u.in.text))
		newIndex, completed, comps, _ := insertComplete(u.in.comps, rw, u.in.index)
		b, err := iorw.ReadFullSlice(rw)
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if !(cmpStrSlices(comps, u.out.comps) &&
			text == u.out.text &&
			newIndex == u.out.newIndex &&
			completed == u.out.completed) {
			t.Fatal(newIndex, completed, comps, text, "expecting", u.out)
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
