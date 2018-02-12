package debug

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
)

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
	if err := enc.Encode(&msg); err != nil {
		return nil, err
	}

	// get bytes
	buf := bbuf.Bytes()

	// encode v size at buffer start
	l := uint32(len(buf) - len(sizeBuf))
	binary.BigEndian.PutUint32(buf, l)

	return buf, nil
}

func DecodeMessage(reader io.Reader) (interface{}, error) {
	// read size
	sizeBuf := make([]byte, 4)
	n, err := reader.Read(sizeBuf)
	if err != nil && !(err == io.EOF && n > 0) {
		return nil, err
	}
	l := binary.BigEndian.Uint32(sizeBuf)

	// read msg
	msgBuf := make([]byte, l)
	n, err = reader.Read(msgBuf)
	if err != nil && !(err == io.EOF && n > 0) {
		return nil, err
	}
	if n != len(msgBuf) {
		return nil, fmt.Errorf("expected to read %v but got %v", len(msgBuf), n)
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
