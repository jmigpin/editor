package toolbarparser

import "testing"

func TestUpdateOrInsertCmd1(t *testing.T) {
	s := "aa|bb| cmd1 cc |dd"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "zz")
	if res.S != "aa|bb| cmd1 zz |dd" {
		t.Fatal(res)
	}
}

// find last cmd part
func TestUpdateOrInsertCmd2(t *testing.T) {
	s := "aa|bb| cmd1 cc | cmd1 cc2 |dd"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "zz")
	if res.S != "aa|bb| cmd1 cc | cmd1 zz |dd" {
		t.Fatal(res)
	}
}

// empty arg given, just get the positions
func TestUpdateOrInsertCmd3(t *testing.T) {
	s := "aa|bb| cmd1 cc | cmd1 cc2 |dd"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "")
	if res.Pos != 22 || res.End != 25 {
		t.Fatal(res)
	}
}

// no cmd present, need to insert
func TestUpdateOrInsertCmd4(t *testing.T) {
	s := "aa|bb|cc|dd"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "zz")
	if res.S != "aa|bb|cc|dd | cmd1 zz" {
		t.Fatal(res)
	}
}

// no cmd present, need to insert
func TestUpdateOrInsertCmd5(t *testing.T) {
	s := "aa|bb|cc|dd|"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "zz")
	if res.S != "aa|bb|cc|dd| cmd1 zz" {
		t.Fatal(res)
	}
}

// no cmd present, need to insert
func TestUpdateOrInsertCmd6(t *testing.T) {
	s := "aa|bb|cc|dd \n   "
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "zz")
	if res.S != "aa|bb|cc|dd \ncmd1 zz" {
		t.Fatal(res)
	}
}

// heading space after cmd
func TestUpdateOrInsertCmd7(t *testing.T) {
	s := "aa|bb|cc| cmd1| ddd"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "zz")
	if res.S != "aa|bb|cc| cmd1 zz| ddd" {
		t.Fatal(res)
	}
}

// heading space after cmd
func TestUpdateOrInsertCmd8(t *testing.T) {
	s := "aa|bb|cc| cmd1| ddd"
	data := Parse(s)
	res := UpdateOrInsertPartCmd(data, "cmd1", "")
	if res.S != "aa|bb|cc| cmd1 | ddd" {
		t.Fatal(res)
	}
}
