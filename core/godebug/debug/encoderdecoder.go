package debug

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"slices"
	"unsafe"
)

var encDecReg = newEncDecRegistry()

//----------
//----------
//----------

// encode/decode, id/type, registry
type EncDecRegistry struct {
	idc      EncDecRegId
	typeToId map[reflect.Type]EncDecRegId
	idToType map[EncDecRegId]reflect.Type

	nilId EncDecRegId
}

func newEncDecRegistry() *EncDecRegistry {
	reg := &EncDecRegistry{}
	reg.typeToId = map[reflect.Type]EncDecRegId{}
	reg.idToType = map[EncDecRegId]reflect.Type{}

	reg.idc = 1                // start at non-zero to detect errors
	reg.nilId = reg.newId(nil) // register nil
	reg.idc = 10               // start other registration at a fixed num

	return reg
}
func (reg *EncDecRegistry) register(v any) EncDecRegId {
	typ := concreteType(reflect.ValueOf(v))
	id, ok := reg.typeToId[typ]
	if ok {
		return id
	}
	id = reg.newId(typ)
	reg.typeToId[typ] = id
	reg.idToType[id] = typ
	return id
}
func (reg *EncDecRegistry) newId(typ reflect.Type) EncDecRegId {
	id := reg.idc
	reg.idc++

	// DEBUG
	//fmt.Printf("reg: %v: %v\n", id, typ)

	return id
}

//----------

type EncDecRegId byte

//----------
//----------
//----------

var encDecHeader = []byte{3, 7} // just to avoid matching random bytes

func encode(w io.Writer, v any, logger Logger) error {
	enc := newEncoder(w, encDecReg)
	enc.W = logger.W
	enc.Prefix = logger.Prefix + enc.Prefix

	// header
	if _, err := enc.w.Write(encDecHeader); err != nil {
		return err
	}

	return enc.reflect(v)
}
func decode(r io.Reader, v any, logger Logger) error {
	dec := newDecoder(r, encDecReg)
	dec.W = logger.W
	dec.Prefix = logger.Prefix + dec.Prefix

	// header
	h := make([]byte, len(encDecHeader))
	if _, err := io.ReadFull(dec.r, h); err != nil {
		return err
	}
	if !slices.Equal(h, encDecHeader) {
		return fmt.Errorf("header mismatch: %v", h)
	}

	return dec.reflect(v)

}

//----------
//----------
//----------

type Encoder struct {
	w   io.Writer
	reg *EncDecRegistry

	Logger
}
type Decoder struct {
	r              io.Reader
	reg            *EncDecRegistry
	firstInterface bool
	firstPointer   bool

	Logger
}

//----------

func newEncoder(w io.Writer, reg *EncDecRegistry) *Encoder {
	enc := &Encoder{w: w, reg: reg}
	enc.Prefix = "enc: "
	return enc
}
func newDecoder(r io.Reader, reg *EncDecRegistry) *Decoder {
	dec := &Decoder{r: r, reg: reg}
	dec.Prefix = "dec: "
	return dec
}

//----------

func (enc *Encoder) sliceLen(n int) error {
	return enc.writeBinary(uint16(n))
}
func (dec *Decoder) sliceLen(v *int) error {
	n := uint16(0)
	err := dec.readBinary(&n)
	*v = int(n)
	return err
}

//----------

func (enc *Encoder) id(id EncDecRegId) error {
	return enc.writeBinary(id)
}
func (dec *Decoder) id() (EncDecRegId, error) {
	id := EncDecRegId(0)
	err := dec.readBinary(&id)
	return id, err
}

//----------

func (enc *Encoder) id2(v reflect.Value) error {
	typ := concreteType(v)
	id, ok := enc.reg.typeToId[typ]
	if !ok {
		return enc.errorf("type has no id: %v", typ)
	}
	return enc.id(id)
}
func (dec *Decoder) id2(id EncDecRegId) (reflect.Type, error) {
	typ, ok := dec.reg.idToType[id]
	if !ok {
		return nil, dec.errorf("id has no type: %v", id)
	}
	return typ, nil
}

//----------

func (enc *Encoder) reflect(v any) error {
	// log encoded bytes at the end
	if enc.W != nil {
		buf := &bytes.Buffer{}
		enc.w = io.MultiWriter(enc.w, buf)
		defer func() {
			enc.logf("encoded byte: %v\n", buf.Bytes())
		}()
	}

	enc.logf("reflect: %T\n", v)

	vv := reflect.ValueOf(v)

	switch vv.Kind() {
	case reflect.Pointer:
	case reflect.Interface:
	default:
		// use interface for other type in order to have an id
		//vv = reflect.ValueOf(vv.Interface())

		return enc.errorf("not a pointer or interface: %T", v)
	}

	return enc.reflect2(vv)
}
func (dec *Decoder) reflect(v any) error {
	dec.logf("%T\n", v)

	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Pointer:
	default:
		return dec.errorf("not a pointer: %T", v)
	}

	if vv.IsNil() {
		vv.Set(reflect.New(vv.Type().Elem()))
	}

	switch vv.Elem().Kind() {
	case reflect.Pointer:
		vv = vv.Elem()
	case reflect.Interface:
		dec.firstInterface = true // allow not knowing the first type
		vv = vv.Elem()
	}

	return dec.reflect2(vv)
}

//----------

func (enc *Encoder) reflect2(v reflect.Value) error {
	enc.logf("reflect2: %v\n", v.Type())

	switch v.Kind() {
	case reflect.Pointer:
		// has always an id because it can be nil
		if v.IsNil() {
			return enc.id(enc.reg.nilId)
		}
		if err := enc.id2(v); err != nil {
			return err
		}

		return enc.reflect2(v.Elem())
	case reflect.Interface:
		// has always an id because it can be nil
		if v.IsNil() {
			return enc.id(enc.reg.nilId)
		}
		if err := enc.id2(v); err != nil {
			return err
		}

		return enc.reflect2(v.Elem())
	case reflect.Struct:
		n := v.NumField()
		vt := v.Type()
		for i := 0; i < n; i++ {
			// embedded fields
			if vt.Field(i).Anonymous {
				continue
			}

			vf := v.Field(i)
			if err := enc.reflect2(vf); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		n := v.Len()
		if err := enc.sliceLen(n); err != nil {
			return err
		}

		// fast path for []byte
		if b, ok := v.Interface().([]byte); ok {
			return enc.writeBinary(b)
		}

		for i := 0; i < n; i++ {
			vi := v.Index(i)
			if err := enc.reflect2(vi); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		u := []byte(v.Interface().(string))
		return enc.reflect2(reflect.ValueOf(u))

	case reflect.Int: // int64, 8 bytes
		u := int64(v.Interface().(int))
		bs := make([]byte, 8)
		binary.BigEndian.PutUint64(bs, uint64(u))
		_, err := enc.w.Write(bs)
		return err

	default:
		return enc.writeBinary(v.Interface())
	}
}
func (dec *Decoder) reflect2(v reflect.Value) error {
	dec.logf("reflect2: %v\n", v.Type())

	switch v.Kind() {
	case reflect.Pointer:
		// handle id
		id, err := dec.id()
		if err != nil {
			return err
		}
		dec.logf("\tpointer id: %v\n", id)
		if id == dec.reg.nilId {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		typ, err := dec.id2(id)
		if err != nil {
			return err
		}

		if !typ.AssignableTo(v.Type().Elem()) {
			return dec.errorf("%v not assignable to %v", typ, v.Type().Elem())
		}

		if v.IsNil() {
			v.Set(reflect.New(typ))
		}

		return dec.reflect2(v.Elem())

	case reflect.Interface:
		// handle id
		id, err := dec.id()
		if err != nil {
			return err
		}
		dec.logf("\tinterface id: %v\n", id)
		if id == dec.reg.nilId {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		typ, err := dec.id2(id)
		if err != nil {
			return err
		}

		if !typ.AssignableTo(v.Type()) {
			return dec.errorf("%s not assignable to %s", typ, v.Type())
		}

		// assign a pointer of the type
		v.Set(reflect.New(typ))

		if dec.firstInterface {
			dec.logf("\tfirstinterface\n")
			dec.firstInterface = false
			v = v.Elem() // bypass the need for a ptr
		}

		return dec.reflect2(v.Elem())

	case reflect.Struct:
		n := v.NumField()
		vt := v.Type()
		for i := 0; i < n; i++ {
			// embedded fields
			if vt.Field(i).Anonymous {
				continue
			}

			vf := v.Field(i)
			if err := dec.reflect2(vf); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice:
		n := 0
		if err := dec.sliceLen(&n); err != nil {
			return err
		}
		dec.logf("\tslice len: %v\n", n)

		// fast path for bytes
		if _, ok := v.Interface().([]byte); ok {
			b := make([]byte, n)
			if _, err := io.ReadFull(dec.r, b); err != nil {
				return err
			}
			v.Set(reflect.ValueOf(b))
			return nil
		}

		v.Set(reflect.MakeSlice(v.Type(), n, n))
		for i := 0; i < n; i++ {
			vi := v.Index(i)
			if err := dec.reflect2(vi); err != nil {
				return err
			}
		}
		return nil
	case reflect.String:
		u := []byte{}
		ut := reflect.ValueOf(&u).Elem()
		if err := dec.reflect2(ut); err != nil {
			return err
		}
		v.Set(reflect.ValueOf(string(u)))
		return nil
	case reflect.Int: // int64, 8 bytes
		ptr := unsafe.Pointer(v.UnsafeAddr())
		_, err := io.ReadFull(dec.r, (*[8]byte)(ptr)[:])
		return err
	default:
		err := dec.readBinary(v.Addr().Interface())
		if err == nil {
			dec.logf("\tbinary: %v\n", v.Interface())
		}
		return nil
	}
}

//----------

func (enc *Encoder) writeBinary(v any) error {
	if err := binary.Write(enc.w, binary.BigEndian, v); err != nil {
		//return enc.errorf("writeBinary(%T): %w", v, err) // DEBUG
		return enc.errorf("write: %w", err) // simpler
	}
	return nil
}
func (dec *Decoder) readBinary(v any) error {
	if err := binary.Read(dec.r, binary.BigEndian, v); err != nil {
		//return dec.errorf("readBinary(%T): %w", v, err) // DEBUG
		return dec.errorf("read: %w", err) // simpler
	}
	return nil
}

//----------
//----------
//----------

func wrapErrorWithType(err error, v any) error {
	if err != nil {
		return fmt.Errorf("%w (%T)", err, v)
	}
	return nil
}

//----------

func concreteType(v reflect.Value) reflect.Type {
	// NOTE: this can't be done directly with reflect.Type because interface type doesn't have a t.elem(), only the interface value does

	for {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				return v.Type().Elem()
			}
			v = v.Elem()
			continue
		case reflect.Interface:
			if v.IsNil() {
				return v.Type()
			}
			v = v.Elem()
			continue
		}
		break
	}
	return v.Type()
}
