package syncutil

import (
	"testing"

	"github.com/jmigpin/editor/util/chanutil"
)

func BenchmarkSyncedQ(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bSyncedQ()
	}
}
func bSyncedQ() {
	sq := NewSyncedQ()
	for i := 0; i < 1000; i++ {
		sq.PushBack(i)
	}
	for i := 0; i < 1000; i++ {
		sq.PopFront()
	}
}

//----------

func BenchmarkChanQ(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bChanQ()
	}
}
func bChanQ() {
	q := chanutil.NewChanQ(16, 16)
	in, out := q.In(), q.Out()
	for i := 0; i < 1000; i++ {
		in <- i
	}
	for i := 0; i < 1000; i++ {
		<-out
	}
}
