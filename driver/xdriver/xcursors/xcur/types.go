package xcur

import (
	"image"
	"time"
)

type Theme struct {
	Name    string
	Cursors map[string]*Cursor
}

type Cursor struct {
	Comments []*Comment
	Images   map[int][]*Image
}

type Image struct {
	NominalSize int
	Delay       time.Duration
	Hot         image.Point
	Bounds      image.Rectangle
	PixARGB     []byte
}

type Comment struct {
	Subtype CommentSubtype
	Comment string
}

type CommentSubtype uint32

const (
	CommentSubtypeCopyright CommentSubtype = 1 + iota
	CommentSubtypeLicense
	CommentSubtypeOther
)

func (c *Cursor) BestSize(size int) (best int) {
	for s := range c.Images {
		best = betterSize(size, best, s)
	}
	return best
}

func betterSize(target, a, b int) int {
	da := dist(target, a)
	db := dist(target, b)
	switch {
	case da < db:
		return a
	case db < da:
		return b
	default:
		return max(a, b)
	}
}

func dist(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}
