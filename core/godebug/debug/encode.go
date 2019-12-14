package debug

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"
)

func RegisterStructure(v interface{}) {
	gob.Register(v)
}

//----------

func EncodeMessage(msg interface{}) ([]byte, error) {
	// message buffer
	var bbuf bytes.Buffer

	// reserve space to encode v size
	sizeBuf := make([]byte, 4)
	if _, err := bbuf.Write(sizeBuf[:]); err != nil {
		return nil, err
	}

	// encode v
	enc := gob.NewEncoder(&bbuf)
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
	var msg interface{}
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
