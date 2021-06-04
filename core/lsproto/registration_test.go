package lsproto

import "testing"

func TestParseRegistration1(t *testing.T) {
	s := "go,.go,tcp,goexec"
	reg, err := NewRegistration(s)
	if err != nil {
		t.Fatal(err)
	}
	s2 := reg.String()
	if s2 != "go,.go,tcp,goexec" {
		t.Fatal(s2)
	}
}

func TestParseRegistration2(t *testing.T) {
	s := "c/c++,.c .h .hpp,tcp,\"cexec opt1\""
	reg, err := NewRegistration(s)
	if err != nil {
		t.Fatal(err)
	}
	s2 := reg.String()
	if s2 != "c/c++,\".c .h .hpp\",tcp,\"cexec opt1\"" {
		t.Fatal(s2)
	}
}

func TestParseRegistration3(t *testing.T) {
	s := "c,.c,tcpclient,127.0.0.1:9000"
	reg, err := NewRegistration(s)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(reg)
	s2 := reg.String()
	if s2 != "c,.c,tcpclient,127.0.0.1:9000" {
		t.Fatal(s2)
	}
}

//----------
