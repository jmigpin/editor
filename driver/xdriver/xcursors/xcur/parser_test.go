package xcur

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	src := makeTestCursorFile()
	cur, err := Decode(src)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(cur.Comments), 1; got != want {
		t.Fatalf("comments: got %v, want %v", got, want)
	}
	if got, want := cur.Comments[0].Comment, "hello"; got != want {
		t.Fatalf("comment: got %q, want %q", got, want)
	}

	imgs := cur.Images[24]
	if got, want := len(imgs), 1; got != want {
		t.Fatalf("images: got %v, want %v", got, want)
	}
	img := imgs[0]
	if got, want := img.Bounds.Dx(), 2; got != want {
		t.Fatalf("width: got %v, want %v", got, want)
	}
	if got, want := img.Bounds.Dy(), 1; got != want {
		t.Fatalf("height: got %v, want %v", got, want)
	}
	if got, want := img.Hot.X, 1; got != want {
		t.Fatalf("hot x: got %v, want %v", got, want)
	}
	if got, want := img.Hot.Y, 0; got != want {
		t.Fatalf("hot y: got %v, want %v", got, want)
	}
	if got, want := img.Delay, 75*time.Millisecond; got != want {
		t.Fatalf("delay: got %v, want %v", got, want)
	}
	if got, want := img.PixARGB, []byte{1, 2, 3, 4, 5, 6, 7, 8}; string(got) != string(want) {
		t.Fatalf("pixels: got %v, want %v", got, want)
	}
}

func makeTestCursorFile() []byte {
	var b []byte
	u32 := func(v uint32) {
		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], v)
		b = append(b, tmp[:]...)
	}

	const (
		headerSize = 16
		ntoc       = 2
		tocSize    = ntoc * 12
		commentPos = headerSize + tocSize
		imagePos   = commentPos + 16 + 4 + len("hello")
	)

	u32(fileMagic)
	u32(headerSize)
	u32(1)
	u32(ntoc)

	u32(tocTypeComment)
	u32(uint32(CommentSubtypeOther))
	u32(commentPos)

	u32(tocTypeImage)
	u32(24)
	u32(uint32(imagePos))

	u32(16)
	u32(tocTypeComment)
	u32(uint32(CommentSubtypeOther))
	u32(1)
	u32(uint32(len("hello")))
	b = append(b, "hello"...)

	u32(16)
	u32(tocTypeImage)
	u32(24)
	u32(1)
	u32(2)
	u32(1)
	u32(1)
	u32(0)
	u32(75)
	b = append(b, []byte{1, 2, 3, 4, 5, 6, 7, 8}...)

	return b
}
