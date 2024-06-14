package debug

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncode1(t *testing.T) {
	lines := OffsetMsgs{
		{Item: IVi(1), FileIndex: 7},
		{Item: IVi(2), FileIndex: 8},
		{Item: IA(nil, 5, nil)},
		//{Item: IA(IL(IVi(3)), 9, IL(IVi(4)))},
		//{Item: IA(nil, 10, IL(IVi(5)))},
	}

	v, _, err := testEncDec(t, &lines)
	if err != nil {
		t.Fatal(err)
	}
	lms, ok := v.(*OffsetMsgs)
	if !ok {
		t.Fatal(v)
	}
	if len(*lms) != len(lines) {
		t.Fatal(v)
	}
	t.Log(lms)
}
func TestEncode2(t *testing.T) {
	/*
		encode
		[
		3 2 7 // header
		16 // offsetmsg id
		0 0 // file index
		0 0 0 0 // debug index
		0 0 0 0 // offset
		15 // itemvalue id (interface)
		15 // itemvalue id (pointer)
		0 1 // str len
		49 // str content
		]
	*/

	lm := &OffsetMsg{Item: IVi(1)}
	v, b, err := testEncDec(t, lm)
	if err != nil {
		t.Fatal(err)
	}

	b2 := "[3 7 16 0 0 0 0 0 0 0 0 0 0 17 17 0 1 49]"
	b3 := fmt.Sprintf("%v", b)
	if b2 != b3 {
		t.Fatalf("expecting:\n%v got\n%v", b2, b3)
	}

	t.Log(v)
}
func TestEncode3(t *testing.T) {
	u := &ReqStartMsg{}
	v, _, err := testEncDec(t, u)
	if err != nil {
		t.Fatal(err)
	}
	exp := "*debug.ReqStartMsg"
	s := fmt.Sprintf("%T", v)
	if s != exp {
		t.Fatal(s)
	}
}

//----------
//----------
//----------

func testEncDec(t *testing.T, v any) (any, []byte, error) {
	logStdout := verboseStdout()
	logger := Logger{"test: ", logStdout}

	buf := &bytes.Buffer{}
	if err := encode(buf, v, logger); err != nil {
		return nil, nil, err
	}
	b := buf.Bytes()

	buf2 := bytes.NewBuffer(b)

	// commented: works as well, but want to support ptr to any
	//res := reflect.New(reflect.TypeOf(v)).Interface()
	//err := decode(buf2, res)

	res := (any)(nil)
	err := decode(buf2, &res, logger)

	return res, b, err
}
