package debug

import (
	"reflect"
	"strconv"
	"strings"
)

func stringify(v interface{}) string {
	return stringifyV3(v)
}
func stringifyV3(v interface{}) string {
	p := newPrint3(150, 7, eso.stringifyBytesRunes)
	p.do(v)
	return p.ToString()
}

//----------
//----------
//----------

// note: avoid using fmt.printf to skip triggering *.String()/*.Error(). Allows debugging string/error funcs.

type print3 struct {
	buf      strings.Builder // need to use strings.builder to construct immutable value to be sent later on
	avail    int
	maxDepth int
	stk      []reflect.Value

	stringifyBytesAndRunes bool
}

func newPrint3(max, maxDepth int, sbr bool) *print3 {
	return &print3{avail: max, maxDepth: maxDepth, stringifyBytesAndRunes: sbr}
}

func (p *print3) do(v interface{}) {
	rv := reflect.ValueOf(v)
	p.doValue(rv, 0)
}
func (p *print3) doValue(v reflect.Value, depth int) {
	defer func() {
		if r := recover(); r != nil {
			// found errors:
			// 	reflect.Value.Interface: cannot return value obtained from unexported field or method
			// 	reflect.Value.UnsafeAddr of unaddressable value
			// 	interface conversion: interface {} is debug.t3, not []uint8}}
			//p.print("PANIC" + fmt.Sprint(r)) // use fmt.sprintf just for debug

			//if err, ok := r.(error); ok {
			//	p.print("<" + err.Error() + ">")
			//}

			p.print("PANIC")
		}
	}()
	p.doValue2(v, depth)
}
func (p *print3) doValue2(v reflect.Value, depth int) {
	//fmt.Printf("dovalue2: %v, %v\n", v.Kind(), v.String())

	p.stk = append(p.stk, v)
	defer func() { p.stk = p.stk[:len(p.stk)-1] }()

	switch v.Kind() {
	case reflect.Struct:
		p.doStruct(v, depth)
	case reflect.Slice, reflect.Array:
		p.doSliceOrArray(v, depth)
	case reflect.Map:
		p.doMap(v, depth)
	case reflect.Pointer:
		p.doPointer(v, depth)
	case reflect.Interface:
		p.doInterface(v, depth)
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		v2 := reflect.ValueOf(v.Pointer())
		p.doValue(v2, depth+1)

	case reflect.Bool:
		p.print(strconv.FormatBool(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.print(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// note: byte=uint8
		p.print(strconv.FormatUint(v.Uint(), 10))
	case reflect.Uintptr:
		//p.print(fmt.Sprintf("%#x", v.Uint())) // #=0x prefix
		p.print("0x" + strconv.FormatUint(v.Uint(), 16))
	case reflect.Float32:
		p.print(strconv.FormatFloat(v.Float(), 'f', -1, 32))
	case reflect.Float64:
		p.print(strconv.FormatFloat(v.Float(), 'f', -1, 64))
	case reflect.Complex64:
		p.print(strconv.FormatComplex(v.Complex(), 'f', -1, 64))
	case reflect.Complex128:
		p.print(strconv.FormatComplex(v.Complex(), 'f', -1, 128))
	case reflect.String:
		p.print("\"")
		defer p.print("\"")
		p.printCut(v.String())
	case reflect.Invalid:
		p.print("nil")
	default:
		p.print("(TODO:")
		defer p.print(")")
		p.print(v.String())
	}
}

//----------

func (p *print3) doPointer(v reflect.Value, depth int) {
	if v.IsNil() {
		p.print("nil")
		return
	}
	p.print("&")
	p.doValue(v.Elem(), depth+1)
}
func (p *print3) doInterface(v reflect.Value, depth int) {
	p.doValue(v.Elem(), depth) // keeping depth to allow more prints
}
func (p *print3) doStruct(v reflect.Value, depth int) {
	p.printStructTypeName(v)

	p.print("{")
	defer p.print("}")
	vt := v.Type()
	for i := 0; i < vt.NumField(); i++ {
		if !p.printLoopSep(i, depth+1) {
			break
		}
		f := v.Field(i)
		p.doValue(f, depth+1)
	}
}
func (p *print3) doSliceOrArray(v reflect.Value, depth int) {
	if p.stringifyBytesAndRunes {
		if p.printSliceOrArrayAsString(v) {
			return
		}
	}

	p.print("[")
	defer p.print("]")
	for i := 0; i < v.Len(); i++ {
		if !p.printLoopSep(i, depth+1) {
			break
		}
		u := v.Index(i)
		p.doValue(u, depth+1)
	}
}
func (p *print3) doMap(v reflect.Value, depth int) {
	p.print("map[")
	defer p.print("]")
	iter := v.MapRange()
	for i := 0; iter.Next(); i++ {
		if !p.printLoopSep(i, depth+1) {
			break
		}
		p.doValue(iter.Key(), depth+1)
		p.print(":")
		p.doValue(iter.Value(), depth+1)
	}
}

//----------

func (p *print3) printStructTypeName(v reflect.Value) {
	//fmt.Printf("printstk\n")
	printType := false
	k := len(p.stk) - 1 - 1 // extra -1 to bypass the struct itself
	for ; k >= 0; k-- {
		v := p.stk[k]
		//fmt.Printf("\tkind %v\n", v.Kind())
		if v.Kind() == reflect.Pointer {
			continue
		}
		if v.Kind() == reflect.Interface {
			printType = true
		}
		break
	}
	if k < 0 { // cover case of interface{} as an arg
		printType = true
	}
	if printType {
		p.print(v.Type().Name())
	}
}

//----------

func (p *print3) printSliceOrArrayAsString(v reflect.Value) bool {
	switch v.Type().Elem().Kind() {
	case reflect.Uint8: // byte
		p.print("\"")
		defer p.print("\"")

		//b := v.Interface().([]byte) // can fail if field unexported
		//b := ReflectValueUnexported(v).Interface().([]byte)
		//p.printBytesCut(b)

		for i := 0; i < v.Len(); i++ {
			if p.avail <= 0 {
				p.print("...")
				break
			}
			u := uint8(v.Index(i).Uint())
			p.printBytes([]byte{u})
		}

		return true
	case reflect.Int32: // rune
		p.print("\"")
		defer p.print("\"")

		//b := v.Interface().([]int32) // can fail if field unexported
		//b := ReflectValueUnexported(v).Interface().([]int32)
		//p.printBytesCut([]byte(string(b)))

		for i := 0; i < v.Len(); i++ {
			if p.avail <= 0 {
				p.print("...")
				break
			}
			u := int32(v.Index(i).Int())
			p.print(string([]rune{u}))
		}

		return true
	}
	return false
}

//----------

func (p *print3) printLoopSep(i int, depth int) bool {
	if depth >= p.maxDepth {
		p.print("...")
		return false
	}
	if i > 0 {
		p.print(" ")
	}
	if p.avail <= 0 {
		p.print("...")
		return false
	}
	return true
}

//----------

func (p *print3) printCut(s string) {
	if len(s) > p.avail {
		p.print(s[:p.avail])
		p.print("...")
		return
	}
	p.print(s)
}
func (p *print3) printBytesCut(b []byte) {
	if len(b) > p.avail {
		p.printBytes(b[:p.avail])
		p.print("...")
		return
	}
	p.printBytes(b)
}

//----------

func (p *print3) print(s string) {
	n, err := p.buf.WriteString(s)
	if err != nil {
		return
	}
	p.avail -= n
}
func (p *print3) printBytes(b []byte) {
	n, err := p.buf.Write(b)
	if err != nil {
		return
	}
	p.avail -= n
}
func (p *print3) canPrint() bool {
	return p.avail >= 0
}
func (p *print3) ToString() string {
	return p.buf.String()
}

//----------
//----------
//----------

//func protect(fn func()) {
//	defer func() {
//		if x := recover(); x != nil {
//			// calling any function here could itself cause a panic in case of "invalid pointer found on stack" (ex: fmt.println)
//		}
//	}()
//	fn()
//}

//func ReflectValueUnexported(v reflect.Value) reflect.Value {
//	if !v.CanAddr() {
//		return v
//	}
//	ptr := unsafe.Pointer(v.UnsafeAddr())
//	return reflect.NewAt(v.Type(), ptr).Elem()
//}

func SprintCutCheckQuote(max int, s string) string {
	if len(s) > max {
		u := s[:max] + "..."
		// close quote if present
		const q = '"'
		if rune(u[0]) == q {
			u += string(q)
		}
		return u
	}
	return s
}
