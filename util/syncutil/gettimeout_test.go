package syncutil

import (
	"testing"
	"time"

	"github.com/jmigpin/editor/util/chanutil"
)

func BenchmarkTimeout(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bTimeout(b)
	}
}
func bTimeout(b *testing.B) {
	for i := 0; i < 1000; i++ {
		u := NewGetTimeout()
		ready := make(chan bool, 1)
		readyFn := func() error { ready <- true; return nil }
		go func() {
			<-ready
			if err := u.Set(i); err != nil {
				b.Log(err)
			}
		}()
		_, err := u.Get(50*time.Millisecond, readyFn)
		if err != nil {
			b.Log(err)
		}
	}
}

//----------

func BenchmarkNBChan(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bNBChan(b)
	}
}
func bNBChan(b *testing.B) {
	for i := 0; i < 1000; i++ {
		u := chanutil.NewNBChan2(1, "")
		go func() {
			if err := u.Send(i); err != nil {
				b.Log(err)
			}
		}()
		_, err := u.Receive(50 * time.Millisecond)
		if err != nil {
			b.Log(err)
		}
	}
}
