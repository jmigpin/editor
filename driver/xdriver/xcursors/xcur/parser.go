package xcur

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"os"
	"time"
)

var ErrBadMagic = errors.New("bad magic")

const (
	fileMagic      = 0x72756358 // ASCII "Xcur"
	tocTypeComment = 0xfffe0001
	tocTypeImage   = 0xfffd0002
)

func DecodeFile(path string) (*Cursor, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return Decode(b)
}

func Decode(src []byte) (*Cursor, error) {
	d := &decoder{src: src}
	return d.parse()
}

type decoder struct {
	src []byte
	pos int
}

type fileToc struct {
	typ      uint32
	subtype  uint32
	position uint32
}

func (d *decoder) parse() (*Cursor, error) {
	cursor := &Cursor{Images: map[int][]*Image{}}

	tocs, err := d.header()
	if err != nil {
		return nil, err
	}

	for _, toc := range tocs {
		if err := d.seek(int(toc.position)); err != nil {
			return nil, err
		}
		toc2, err := d.tocHeader()
		if err != nil {
			return nil, err
		}
		if toc2.typ != toc.typ || toc2.subtype != toc.subtype {
			return nil, fmt.Errorf("toc mismatch at %v", toc.position)
		}

		switch toc.typ {
		case tocTypeComment:
			comment, err := d.comment(toc)
			if err != nil {
				return nil, err
			}
			cursor.Comments = append(cursor.Comments, comment)
		case tocTypeImage:
			img, err := d.image(toc)
			if err != nil {
				return nil, err
			}
			cursor.Images[img.NominalSize] = append(cursor.Images[img.NominalSize], img)
		default:
			return nil, fmt.Errorf("unknown toc type: 0x%x", toc.typ)
		}
	}
	return cursor, nil
}

func (d *decoder) header() ([]fileToc, error) {
	magic, err := d.u32le()
	if err != nil {
		return nil, err
	}
	if magic != fileMagic {
		return nil, ErrBadMagic
	}

	if _, err := d.u32le(); err != nil { // header size
		return nil, err
	}
	if _, err := d.u32le(); err != nil { // version
		return nil, err
	}
	ntoc, err := d.u32le()
	if err != nil {
		return nil, err
	}

	tocs := make([]fileToc, 0, ntoc)
	for range ntoc {
		toc, err := d.tocEntry()
		if err != nil {
			return nil, err
		}
		tocs = append(tocs, toc)
	}
	return tocs, nil
}

func (d *decoder) tocEntry() (fileToc, error) {
	typ, err := d.u32le()
	if err != nil {
		return fileToc{}, err
	}
	subtype, err := d.u32le()
	if err != nil {
		return fileToc{}, err
	}
	position, err := d.u32le()
	if err != nil {
		return fileToc{}, err
	}
	return fileToc{typ: typ, subtype: subtype, position: position}, nil
}

func (d *decoder) tocHeader() (fileToc, error) {
	if _, err := d.u32le(); err != nil { // header size
		return fileToc{}, err
	}
	typ, err := d.u32le()
	if err != nil {
		return fileToc{}, err
	}
	subtype, err := d.u32le()
	if err != nil {
		return fileToc{}, err
	}
	if _, err := d.u32le(); err != nil { // version
		return fileToc{}, err
	}
	return fileToc{typ: typ, subtype: subtype}, nil
}

func (d *decoder) comment(toc fileToc) (*Comment, error) {
	n, err := d.u32le()
	if err != nil {
		return nil, err
	}
	b, err := d.bytesN(int(n))
	if err != nil {
		return nil, err
	}
	return &Comment{
		Subtype: CommentSubtype(toc.subtype),
		Comment: string(b),
	}, nil
}

func (d *decoder) image(toc fileToc) (*Image, error) {
	w, err := d.u32le()
	if err != nil {
		return nil, err
	}
	h, err := d.u32le()
	if err != nil {
		return nil, err
	}
	xhot, err := d.u32le()
	if err != nil {
		return nil, err
	}
	yhot, err := d.u32le()
	if err != nil {
		return nil, err
	}
	delay, err := d.u32le()
	if err != nil {
		return nil, err
	}
	if w > 1<<15 || h > 1<<15 {
		return nil, fmt.Errorf("image too large: %vx%v", w, h)
	}
	n := int(w) * int(h) * 4
	pixels, err := d.bytesN(n)
	if err != nil {
		return nil, err
	}
	img := &Image{
		NominalSize: int(toc.subtype),
		Delay:       time.Duration(delay) * time.Millisecond,
		Hot:         image.Pt(int(xhot), int(yhot)),
		Bounds:      image.Rect(0, 0, int(w), int(h)),
		PixARGB:     pixels,
	}
	return img, nil
}

func (d *decoder) seek(pos int) error {
	if pos < 0 || pos > len(d.src) {
		return fmt.Errorf("seek outside source: %v", pos)
	}
	d.pos = pos
	return nil
}

func (d *decoder) u32le() (uint32, error) {
	b, err := d.bytesN(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (d *decoder) bytesN(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative byte count: %v", n)
	}
	pos2 := d.pos + n
	if pos2 < d.pos || pos2 > len(d.src) {
		return nil, fmt.Errorf("short read at offset %v: need %v bytes", d.pos, n)
	}
	b := append([]byte(nil), d.src[d.pos:pos2]...)
	d.pos = pos2
	return b, nil
}
