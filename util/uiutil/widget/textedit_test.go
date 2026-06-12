package widget

import (
	"testing"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func TestStableCursor(t *testing.T) {
	content := "line 1\nline 2 with emojis 💖😀\nline 3\nline 4\n"
	rw := iorw.NewStringReaderAt(content)

	testCases := []struct {
		name string
		idx  int
	}{
		{"start", 0},
		{"middle of line 2 before emoji", 13},
		{"middle of line 2 inside/after emoji", 30},
		{"end", len(content)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos := GetStableCursorPos(rw, tc.idx)
			newIdx := FindStableCursorIndex(rw, pos)
			if newIdx != tc.idx {
				t.Fatalf("expected index %d, got %d for case %s", tc.idx, newIdx, tc.name)
			}
		})
	}
}

func TestStableCursorLargeFile(t *testing.T) {
	var content string
	for i := 0; i < 100; i++ {
		content += "this is line number which has some length to build a large file\n"
	}
	content += "target line with emojis 🌟✨ here\n"
	content += "final line\n"

	rw := iorw.NewStringReaderAt(content)
	idx := len(content) - 15

	pos := GetStableCursorPos(rw, idx)
	if pos.Offset == 0 {
		t.Fatalf("expected offset to be greater than 0 for large file, got %d", pos.Offset)
	}

	newIdx := FindStableCursorIndex(rw, pos)
	if newIdx != idx {
		t.Fatalf("expected index %d, got %d after mapping", idx, newIdx)
	}
}

func TestStableCursorFormatter(t *testing.T) {
	// Original content: spaces + characters + newline
	spaces := ""
	for i := 0; i < 10; i++ {
		spaces += "\t"
	}
	chars := "1234567890" // 50 chars
	content1 := spaces + chars
	idx1 := len(content1)
	newlineAndExtra := "\nabcdef\n----\n"
	content1 += newlineAndExtra

	rw1 := iorw.NewStringReaderAt(content1)
	// Cursor at the last character before newline (index)

	pos := GetStableCursorPos(rw1, idx1)
	t.Logf("Saved position: Offset=%d, Line=%d, Col=%d", pos.Offset, pos.Line, pos.Col)

	// Formatted content: leading spaces removed
	idx2 := len(chars)
	content2 := chars + newlineAndExtra
	rw2 := iorw.NewStringReaderAt(content2)

	newIdx := FindStableCursorIndex(rw2, pos)
	if newIdx != idx2 {
		t.Fatalf("expected restored index %d, got %d (rune %q)", idx2, newIdx, string(content2[newIdx]))
	}
}

func TestStableCursorStartOfLine(t *testing.T) {
	chars := "12345678901234567890123456789012345678901234567890"
	content1 := chars + "\n"
	rw1 := iorw.NewStringReaderAt(content1)
	idx1 := 0

	pos := GetStableCursorPos(rw1, idx1)

	content2 := "    " + chars + "\n"
	rw2 := iorw.NewStringReaderAt(content2)

	newIdx := FindStableCursorIndex(rw2, pos)
	if newIdx != 0 {
		t.Fatalf("expected restored index 0 (start of line), got %d", newIdx)
	}
}


