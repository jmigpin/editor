package tautil

import (
	//"fmt"
	"testing"
)

type TextaTester struct {
	Texta
	str            string
	cursorIndex    int
	selectionIndex int
	selectionOn    bool
}

func (ta *TextaTester) Str() string {
	return ta.str
}
func (ta *TextaTester) CursorIndex() int {
	return ta.cursorIndex
}
func (ta *TextaTester) SetCursorIndex(v int) {
	if v < 0 {
		v = 0
	} else if v > len(ta.str) {
		v = len(ta.str)
	}
	ta.cursorIndex = v
}
func (ta *TextaTester) SelectionIndex() int {
	return ta.selectionIndex
}
func (ta *TextaTester) SetSelectionIndex(v int) {
	if v < 0 {
		v = 0
	} else if v > len(ta.str) {
		v = len(ta.str)
	}
	ta.selectionIndex = v
}
func (ta *TextaTester) SetSelectionOn(v bool) {
	ta.selectionOn = v
}
func (ta *TextaTester) SelectionOn() bool {
	visible := ta.CursorIndex() != ta.SelectionIndex()
	return ta.selectionOn && visible
}
func (ta *TextaTester) EditOpen() {
}
func (ta *TextaTester) EditInsert(index int, str string) {
	ta.str = ta.str[:index] + str + ta.str[index:]
}
func (ta *TextaTester) EditDelete(index, index2 int) {
	ta.str = ta.str[:index] + ta.str[index2:]
}
func (ta *TextaTester) EditClose() {
	ta.SetCursorIndex(ta.CursorIndex())
	ta.SetSelectionIndex(ta.SelectionIndex())
}

func TestMoveCursorJump0(t *testing.T) {
	str := "abcd\n abcd\nabcd"
	ta := &TextaTester{
		str:         str,
		cursorIndex: 0,
	}
	MoveCursorJumpRight(ta, false)
	if !(ta.CursorIndex() == 4) {
		t.Fatal(ta.CursorIndex())
	}
	MoveCursorJumpRight(ta, false)
	if !(ta.CursorIndex() == 6) {
		t.Fatal(ta.CursorIndex())
	}
}
func TestMoveCursorJump1(t *testing.T) {
	str := " abcde abcde "
	ta := &TextaTester{
		str:            str,
		cursorIndex:    3,
		selectionIndex: 3,
	}
	MoveCursorJumpRight(ta, false)
	if !(ta.CursorIndex() == 6 && ta.SelectionIndex() == 3) {
		t.Fatal("t1", ta.CursorIndex(), ta.SelectionIndex())
	}
	MoveCursorJumpRight(ta, false)
	if !(ta.CursorIndex() == 7 && ta.SelectionIndex() == 3) {
		t.Fatal("t2", ta.CursorIndex(), ta.SelectionIndex())
	}
	MoveCursorJumpRight(ta, false)
	if !(ta.CursorIndex() == 12 && ta.SelectionIndex() == 3) {
		t.Fatal("t3", ta.CursorIndex(), ta.SelectionIndex())
	}
}

//func testTabLeft(t *testing.T, str1 string, ci1, si1 int, sOn bool, str2 string, ci2, si2 int) {
//ta := &TextaTester{
//str:            str1,
//cursorIndex:    ci1,
//selectionIndex: si1,
//selectionOn:    sOn,
//}
//TabLeft(ta)
//if !(ta.str == str2 &&
//ta.cursorIndex == ci2 &&
//ta.selectionIndex == si2) {
//t.Fatalf("%+v", ta)
//}
//}

//func TestTabLeft0(t *testing.T) {
//s1 := "	\n	abcd"
//s2 := "	\nabcd"
//testTabLeft(t, s1, 7, 7, false, s2, 6, 6)
//testTabLeft(t, s1, 7, 1, false, s2, 6, 0)
//}
//func TestTabLeft1(t *testing.T) {
//s1 := "	\n	abcd"
//s2 := "\nabcd"
//testTabLeft(t, s1, 7, 1, true, s2, 5, 0)
//}
//func TestTabLeft1_2(t *testing.T) {
//s1 := "		\n	abcd"
//s2 := "	\nabcd"
//testTabLeft(t, s1, 8, 2, true, s2, 6, 1)
//}
//func TestTabLeft2(t *testing.T) {
//s1 := "	\n	\n	abcd"
//s2 := "	\n\nabcd"
//testTabLeft(t, s1, 9, 2, true, s2, 7, 2)
//}
//func TestTabLeft3(t *testing.T) {
//s1 := "	\n	abcd"
//s2 := "	\nabcd"
//testTabLeft(t, s1, 2, 0, false, s2, 2, 0)
//}
//func TestTabLeft4(t *testing.T) {
//s1 := "	\n	abcd"
//s2 := "\nabcd"
//testTabLeft(t, s1, 2, 1, true, s2, 1, 0)
//}
