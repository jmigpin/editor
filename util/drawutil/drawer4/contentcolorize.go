package drawer4

import (
	"bytes"
	"image/color"

	"github.com/jmigpin/editor/util/iout/iorw"
)

func updateContentColorizeOps(d *Drawer) {
	if contentColorizeDone(d) {
		return
	}
	d.Opt.ContentColorize.Group.Ops = contentColorizeOps(d)
}

func contentColorizeDone(d *Drawer) bool {
	if !d.Opt.ContentColorize.Git.On {
		d.Opt.ContentColorize.Group.Ops = nil
		return true
	}
	if d.opt.contentColorize.updated {
		return true
	}
	d.opt.contentColorize.updated = true
	return false
}

func contentColorizeOps(d *Drawer) []*ColorizeOp {
	if d.Opt.ContentColorize.Git.On {
		return gitColorizeOps(d)
	}
	return nil
}

//----------

func gitColorizeOps(d *Drawer) []*ColorizeOp {
	o, n, _, _ := d.visibleLen()
	min, max := o, o+n

	//o := d.opt.runeO.offset
	//min, err := iorw.LineStartIndex(d.reader, o)
	//if err != nil {
	//	return nil
	//}
	//max := min + contentColorizeReadLimit
	//if max > d.reader.Max() {
	//	max = d.reader.Max()
	//}

	r := iorw.NewLimitedReaderAt(d.reader, min, max)
	src, err := iorw.ReadFastFull(r)
	if err != nil {
		return nil
	}
	base := r.Min()

	var ops []*ColorizeOp
	for start := 0; start < len(src); {
		end := bytes.IndexByte(src[start:], '\n')
		lineEnd := len(src)
		next := len(src)
		if end >= 0 {
			lineEnd = start + end
			next = lineEnd + 1
		}

		if start < lineEnd {
			fg := gitLineFg(d, src[start:lineEnd])
			if fg != nil {
				ops = append(ops,
					&ColorizeOp{Offset: base + start, Fg: fg},
					&ColorizeOp{Offset: base + lineEnd},
				)
			}
		}

		start = next
	}
	return ops
}

func gitLineFg(d *Drawer, line []byte) color.Color {
	switch line[0] {
	case '+':
		return d.Opt.ContentColorize.Git.AddFg
	case '-':
		return d.Opt.ContentColorize.Git.DeleteFg
	}
	return nil
}

const contentColorizeReadLimit = 64 * 1024
