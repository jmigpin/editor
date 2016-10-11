package tautil

import (
	//"fmt"
	"testing"
)

func TestMoveCursorJump0(t *testing.T) {
	text := "abcd\n abcd\nabcd"
	ta := &Textad{
		text:        text,
		cursorIndex: 0,
	}
	MoveCursorJumpRight(ta)
	if !(ta.CursorIndex() == 4) {
		t.Fatal(ta.CursorIndex())
	}
	MoveCursorJumpRight(ta)
	if !(ta.CursorIndex() == 6) {
		t.Fatal(ta.CursorIndex())
	}
}
func TestMoveCursorJump1(t *testing.T) {
	text := " abcde abcde "
	ta := &Textad{
		text:        text,
		cursorIndex: 3,
		//selectionOn:    true,
		selectionIndex: 3,
	}
	MoveCursorJumpRight(ta)
	if !(ta.CursorIndex() == 6 && ta.SelectionIndex() == 3) {
		t.Fatal("t1", ta.CursorIndex(), ta.SelectionIndex())
	}
	MoveCursorJumpRight(ta)
	if !(ta.CursorIndex() == 7 && ta.SelectionIndex() == 3) {
		t.Fatal("t2", ta.CursorIndex(), ta.SelectionIndex())
	}
	MoveCursorJumpRight(ta)
	if !(ta.CursorIndex() == 12 && ta.SelectionIndex() == 3) {
		t.Fatal("t3", ta.CursorIndex(), ta.SelectionIndex())
	}
}
