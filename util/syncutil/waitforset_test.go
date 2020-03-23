package syncutil

import (
	"testing"
	"time"

	"github.com/jmigpin/editor/util/chanutil"
)

func BenchmarkWaitForSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		waitForSet1(b)
	}
}
func waitForSet1(b *testing.B) {
	for i := 0; i < 1000; i++ {
		u := NewWaitForSet()
		u.Start(50 * time.Millisecond)
		go func() {
			if err := u.Set(i); err != nil {
				b.Log(err)
			}
		}()
		_, err := u.WaitForSet()
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
