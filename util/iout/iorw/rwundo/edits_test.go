package rwundo

import (
	"testing"
)

func TestEdits1(t *testing.T) {
	ur := &UndoRedo{Index: 10, D: []byte("0"), I: []byte("12")}
	edits := &Edits{}
	edits.Append(ur)

	a, b, ok := edits.preCursor.SelectionIndexesUnsorted()
	if !(ok == true && a == 10 && b == 11) {
		t.Fatal()
	}

	a, b, ok = edits.postCursor.SelectionIndexesUnsorted()
	if !(ok == true && a == 10 && b == 12) {
		t.Fatal()
	}
}

func TestEdits2(t *testing.T) {
	ur := &UndoRedo{Index: 10, D: []byte(""), I: []byte("12")}
	edits := &Edits{}
	edits.Append(ur)

	ok := edits.preCursor.HaveSelection()
	b := edits.preCursor.Index()
	if !(ok == false && b == 10) {
		t.Fatal()
	}

	a, b, ok := edits.postCursor.SelectionIndexesUnsorted()
	if !(ok == true && a == 10 && b == 12) {
		t.Fatal()
	}
}

func TestEdits3(t *testing.T) {
	ur := &UndoRedo{Index: 10, D: []byte("0"), I: []byte("")}
	edits := &Edits{}
	edits.Append(ur)

	a, b, ok := edits.preCursor.SelectionIndexesUnsorted()
	_ = a
	if !(ok == true && b == 11) {
		t.Fatal()
	}

	ok = edits.postCursor.HaveSelection()
	b = edits.postCursor.Index()
	if !(ok == false && b == 10) {
		t.Fatal()
	}
}
