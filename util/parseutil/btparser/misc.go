package btparser

import (
	"iter"
	"reflect"
	"unsafe"
)

func Debug2(fn func()) MFn {
	return func(ps *ParserState, pos Pos) (MPos, error) {
		fn()
		return MPos{pos, pos}, nil
	}
}

//----------

func BytesSnippet(src []byte, mp MPos, pad int) string {
	start, end := mp.Bounds()
	mp = MPos{Start: start, End: end}

	// pad n in each direction for error string
	i1 := max(mp.Start-Pos(pad), 0)
	i2 := min(mp.End+Pos(pad), Pos(len(src)))
	mp.Start -= i1
	mp.End -= i1
	mp.End = min(mp.End, Pos(len(src)))

	// TODO: ensure the total length is small

	s := string(src[i1:i2])
	if s == "" {
		return ""
	}

	insert := func(p Pos, u string) {
		s = s[:p] + u + s[p:]
	}
	// make insertions, from end to start
	if int(i2) < len(src)-1 {
		insert(Pos(len(s)), "...")
	}
	if mp.End-mp.Start >= 2 {
		insert(mp.End, "◄")
		//insert(mp.End, "◙")
		//insert(mp.End, "○")
	}
	insert(mp.Start, "●")
	if i1 > 0 {
		insert(0, "...")
	}
	return s
}

//----------

// NOTE: use this to check if the value is nil inside the iteration
//for name, ptr := range IterateStructFields(ro, false) {
//	if ptr == nil || reflect.ValueOf(ptr).IsNil() {
//		continue
//	}
//	...
//}

func IterateStructFields(tag string, v any, byPtr bool) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		val := reflect.ValueOf(v)
		if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
			return
		}
		val = val.Elem()
		typ := val.Type()

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			tag := field.Tag.Get(tag)
			if tag == "" {
				continue
			}
			fv := val.Field(i)
			u := (any)(nil)
			if byPtr {
				h := unsafe.Pointer(fv.UnsafeAddr())
				u = reflect.NewAt(fv.Type(), h).Interface()
			} else {
				u = fv.Interface()
			}
			if !yield(tag, u) {
				return
			}
		}
	}
}
