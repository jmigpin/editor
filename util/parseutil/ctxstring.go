package parseutil

import (
	"fmt"

	"github.com/jmigpin/editor/util/iout/iorw"
	"github.com/jmigpin/editor/util/mathutil"
)

func CtxErrorStr(rd iorw.ReaderAt, filename string, pos int, msg string, contextSize int) string {
	s, err := CtxString(rd, pos, contextSize)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}
	return fmt.Sprintf("%v:%v: %s: %q", filename, pos, msg, s)
}
func CtxString(rd iorw.ReaderAt, pos int, contextSize int) (string, error) {
	// pad n in each direction for error string
	pad := contextSize / 2
	i := mathutil.Max(pos-pad, 0)
	i2 := mathutil.Min(pos+pad, rd.Max())

	// read src string
	b, err := rd.ReadFastAt(i, i2-i)
	if err != nil {
		return "", fmt.Errorf("ctxstring: failed to get src: %w", err)
	}
	src := string(b)

	// position indicator
	c := pos - i

	sep := "●" // "←"
	src = src[:c] + sep + src[c:]
	if i > 0 {
		src = "..." + src
	}
	if i2 < rd.Max()-1 {
		src = src + "..."
	}
	return src, nil
}
