package debug

import (
	"fmt"
	"reflect"
	"strconv"
)

func stringifyV(v V) string {
	//return stringifyV1(v)
	return stringifyV2(v)
}

//----------

func stringifyV1(v V) string {
	// Note: rune is an alias for int32, can't "case rune:"
	const max = 150
	qFmt := limitFormat(max, "%q")
	str := ""
	switch t := v.(type) {
	case nil:
		return "nil"
	case error:
		str = ReducedSprintf(max, qFmt, t)
	case string:
		str = ReducedSprintf(max, qFmt, t)
	case []string:
		str = quotedStrings(max, t)
	case fmt.Stringer:
		str = ReducedSprintf(max, qFmt, t)
	case []byte:
		str = ReducedSprintf(max, qFmt, t)
	case float32:
		str = strconv.FormatFloat(float64(t), 'f', -1, 32)
	case float64:
		str = strconv.FormatFloat(t, 'f', -1, 64)
	default:
		u := limitFormat(max, "%v")
		str = ReducedSprintf(max, u, v) // ex: bool
	}
	return str
}

//----------

func ReducedSprintf(max int, format string, a ...interface{}) string {
	w := NewLimitedWriter(max)
	_, err := fmt.Fprintf(w, format, a...)
	s := string(w.Bytes())
	if err == LimitReachedErr {
		s += "..."
		// close quote if present
		const q = '"'
		if rune(s[0]) == q {
			s += string(q)
		}
	}
	return s
}

func quotedStrings(max int, a []string) string {
	w := NewLimitedWriter(max)
	sp := ""
	limited := 0
	uFmt := limitFormat(max, "%s%q")
	for i, s := range a {
		if i > 0 {
			sp = " "
		}
		n, err := fmt.Fprintf(w, uFmt, sp, s)
		if err != nil {
			if err == LimitReachedErr {
				limited = n
			}
			break
		}
	}
	s := string(w.Bytes())
	if limited > 0 {
		s += "..."
		if limited >= 2 { // 1=space, 2=quote
			s += `"` // close quote
		}
	}
	return "[" + s + "]"
}

func limitFormat(max int, s string) string {
	// not working: attempt to speedup by using max width (performance)
	//s = strings.ReplaceAll(s, "%", fmt.Sprintf("%%.%d", max))
	return s
}

//----------
//----------
//----------

func stringifyV2(v interface{}) string {
	p := &Print{Max: 150}
	p.Do(v)
	return string(p.Out)
}

//----------

type Print struct {
	Max int // not a strict max, it helps decide to reduce ouput
	Out []byte
}

func (p *Print) Do(v interface{}) {
	ctx := &Ctx{}
	ctx = ctx.WithInInterface(0)
	p.do(ctx, v, 0)
}

func (p *Print) do(ctx *Ctx, v interface{}, depth int) {
	switch t := v.(type) {
	case nil:
		p.appendStr("nil")
	case bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		complex64, complex128:
		s := fmt.Sprintf("%v", t)
		p.appendStr(s)
	case float32:
		s := strconv.FormatFloat(float64(t), 'f', -1, 32)
		p.appendStr(s)
	case float64:
		s := strconv.FormatFloat(t, 'f', -1, 64)
		p.appendStr(s)
	case string:
		p.appendStrQuoted(p.limitStr(t))
	case []byte:
		p.appendBytes(p.limitBytes(t))
	case uintptr:
		if t == 0 {
			p.do(ctx, nil, depth)
			return
		}
		p.appendStr(fmt.Sprintf("%#x", t))

	case error:
		p.appendStrQuoted(p.limitStr(t.Error())) // TODO: big output
	case fmt.Stringer:
		p.appendStrQuoted(p.limitStr(t.String())) // TODO: big output
	default:
		p.doValue(ctx, reflect.ValueOf(v), depth)
	}
}

func (p *Print) doValue(ctx *Ctx, v reflect.Value, depth int) {
	switch v.Kind() {
	case reflect.Bool:
		p.do(ctx, v.Bool(), depth)
	case reflect.String:
		p.do(ctx, v.String(), depth)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.do(ctx, v.Int(), depth)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		p.do(ctx, v.Uint(), depth)
	case reflect.Float32,
		reflect.Float64:
		p.do(ctx, v.Float(), depth)
	case reflect.Complex64,
		reflect.Complex128:
		p.do(ctx, v.Complex(), depth)

	case reflect.Ptr:
		p.doPointer(ctx, v, depth)
	case reflect.Struct:
		p.appendStr("{")
		p.doStruct(ctx, v, depth)
		p.appendStr("}")
	case reflect.Map:
		p.appendStr("map[")
		p.doMap(ctx, v, depth)
		p.appendStr("]")
	case reflect.Slice, reflect.Array:
		p.appendStr("[")
		p.doSlice(ctx, v, depth)
		p.appendStr("]")
	case reflect.Interface:
		p.doInterface(ctx, v, depth)
	case reflect.Chan,
		reflect.Func,
		reflect.UnsafePointer:
		p.do(ctx, v.Pointer(), depth)
	default:
		s := fmt.Sprintf("(todo:%v)", v.Type().String())
		p.appendStr(s)
	}
}

//----------

func (p *Print) doPointer(ctx *Ctx, v reflect.Value, depth int) {
	if depth >= 3 || v.Pointer() == 0 {
		p.do(ctx, v.Pointer(), depth)
		return
	}

	p.appendStr("&")
	e := v.Elem()

	// type name if in interface ctx
	if ctx.ValueInInterface(depth) {
		switch e.Kind() {
		case reflect.Struct:
			p.appendStr(e.Type().Name())
		case reflect.Ptr:
			ctx = ctx.WithInInterface(depth + 1)
		}
	}

	p.doValue(ctx, e, depth+1)
}

func (p *Print) doStruct(ctx *Ctx, v reflect.Value, depth int) {
	//ctx = ctx.WithInStruct(depth + 1)
	vt := v.Type()
	for i := 0; i < vt.NumField(); i++ {
		f := v.Field(i)
		if i > 0 {
			p.appendStr(" ")
		}
		if p.maxedOut() {
			p.appendStr("...")
			break
		}
		p.doValue(ctx, f, depth+1)
	}
}

func (p *Print) doMap(ctx *Ctx, v reflect.Value, depth int) {
	iter := v.MapRange()
	for i := 0; iter.Next(); i++ {
		if i > 0 {
			p.appendStr(" ")
		}
		if p.maxedOut() {
			p.appendStr("...")
			break
		}
		p.doValue(ctx, iter.Key(), depth+1)
		p.appendStr(":")
		p.doValue(ctx, iter.Value(), depth+1)
	}
}

func (p *Print) doSlice(ctx *Ctx, v reflect.Value, depth int) {
	for i := 0; i < v.Len(); i++ {
		u := v.Index(i)
		if i > 0 {
			p.appendStr(" ")
		}
		if p.maxedOut() {
			p.appendStr("...")
			break
		}
		p.doValue(ctx, u, depth+1)
	}
}

func (p *Print) doInterface(ctx *Ctx, v reflect.Value, depth int) {
	e := v.Elem()
	if !e.IsValid() {
		p.appendStr("nil")
		return
	}

	if e.Kind() == reflect.Struct {
		p.appendStr(e.Type().Name())
	}

	ctx = ctx.WithInInterface(depth + 1)
	p.doValue(ctx, e, depth+1)
}

//----------

func (p *Print) maxedOut() bool {
	return p.Max-len(p.Out) <= 0
}

func (p *Print) limitStr(s string) string {
	max := p.Max - len(p.Out)
	if max < 0 {
		max = 0
	}
	if len(s) > 0 && len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func (p *Print) limitBytes(s []byte) []byte {
	return []byte(p.limitStr(string(s)))
}

func (p *Print) appendStrQuoted(s string) {
	p.appendStr(strconv.Quote(s))
}

func (p *Print) appendStr(s string) {
	p.Out = append(p.Out, []byte(s)...)
}
func (p *Print) appendBytes(s []byte) {
	p.Out = append(p.Out, s...)
}

//----------

type Ctx struct {
	Parent *Ctx
	// name/value (short names to avoid usage, still exporting it)
	N string
	V interface{}
}

func (ctx *Ctx) WithValue(name string, value interface{}) *Ctx {
	return &Ctx{ctx, name, value}
}

func (ctx *Ctx) Value(name string) (interface{}, *Ctx) {
	for c := ctx; c != nil; c = c.Parent {
		if c.N == name {
			return c.V, c
		}
	}
	return nil, nil
}

//----------

func (ctx *Ctx) ValueBool(name string) bool {
	v, _ := ctx.Value(name)
	if v == nil {
		return false
	}
	return v.(bool)
}

func (ctx *Ctx) ValueIntM1(name string) int {
	v, _ := ctx.Value(name)
	if v == nil {
		return -1
	}
	return v.(int)
}

//----------

func (ctx *Ctx) WithInInterface(depth int) *Ctx {
	return ctx.WithValue("in_interface_depth", depth)
}
func (ctx *Ctx) ValueInInterface(depth int) bool {
	return ctx.ValueIntM1("in_interface_depth") == depth
}

//----------

//func (ctx *Ctx) WithInStruct(depth int) *Ctx {
//	return ctx.WithValue("in_struct_depth", depth)
//}
//func (ctx *Ctx) ValueInStruct(depth int) bool {
//	return ctx.ValueIntM1("in_struct_depth") == depth
//}
