package debug

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"reflect"
)

func registerForEncodeDecode(encoderId string, v interface{}) {
	// commented: needs encoderId to avoid name clashes when self debugging
	//gob.Register(v)

	rt := reflect.TypeOf(v)
	name := rt.String() // ex: *debug.ReqFilesDataMsg

	// after: rt = rt.Elem()
	// 	rt.Name() // ex: ReqFilesDataMsg
	// 	rt.PkgPath() // ex: github.com/jmigpin/editor/core/godebug/debug
	// 	rt.PkgPath() // ex: godebugconfig/debug

	s := fmt.Sprintf("%v:%v", encoderId, name)
	gob.RegisterName(s, v)
}

//----------

func EncodeMessage(msg interface{}) ([]byte, error) {
	// message buffer
	bbuf := &bytes.Buffer{}

	// reserve space to encode v size
	sizeBuf := make([]byte, 4)
	if _, err := bbuf.Write(sizeBuf[:]); err != nil {
		return nil, err
	}

	// encode v
	enc := gob.NewEncoder(bbuf)
	if err := enc.Encode(&msg); err != nil { // decoder uses &interface{}
		return nil, err
	}

	// get bytes
	buf := bbuf.Bytes()

	// encode v size at buffer start
	l := uint32(len(buf) - len(sizeBuf))
	binary.BigEndian.PutUint32(buf, l)

	return buf, nil
}

func DecodeMessage(rd io.Reader) (interface{}, error) {
	// read size
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(rd, sizeBuf); err != nil {
		return nil, err
	}
	l := int(binary.BigEndian.Uint32(sizeBuf))

	// read msg
	msgBuf := make([]byte, l)
	if _, err := io.ReadFull(rd, msgBuf); err != nil {
		return nil, err
	}

	// decode msg
	buf := bytes.NewBuffer(msgBuf)
	dec := gob.NewDecoder(buf)
	msg := interface{}(nil)
	if err := dec.Decode(&msg); err != nil {
		return nil, err
	}

	return msg, nil
}

//----------

// TODO: document why this simplified version doesn't work (hangs)

//func EncodeMessage(msg interface{}) ([]byte, error) {
//	var buf bytes.Buffer
//	enc := gob.NewEncoder(&buf)
//	if err := enc.Encode(&msg); err != nil {
//		return nil, err
//	}
//	return buf.Bytes(), nil
//}

//func DecodeMessage(reader io.Reader) (interface{}, error) {
//	dec := gob.NewDecoder(reader)
//	var msg interface{}
//	if err := dec.Decode(&msg); err != nil {
//		return nil, err
//	}
//	return msg, nil
//}

//----------
