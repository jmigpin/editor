package fswatcher

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

//----------

func tmpDir() string {
	name, err := ioutil.TempDir("", "watcher_test")
	if err != nil {
		panic(err)
	}
	return name
}

func mustMkdirAll(t *testing.T, name string) {
	err := os.MkdirAll(name, 0755)
	if err != nil {
		t.Fatal(err)
	}
}

func mustRemoveAll(t *testing.T, name string) {
	t.Helper()
	if err := os.RemoveAll(name); err != nil {
		t.Fatal(err)
	}
}

func mustRenameFile(t *testing.T, name, name2 string) {
	t.Helper()
	if err := os.Rename(name, name2); err != nil {
		t.Fatal(err)
	}
}

func mustCreateFile(t *testing.T, name string) {
	t.Helper()
	f, err := os.OpenFile(name, os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, name string) {
	t.Helper()
	f, err := os.OpenFile(name, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte(fmt.Sprintf("%v", rand.Int()))

	n, err := f.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if n < len(data) {
		t.Fatal("short write")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

//----------

func mustAddWatch(t *testing.T, w Watcher, name string) {
	t.Helper()
	if err := w.Add(name); err != nil {
		t.Fatal(err)
	}
}

func mustRemoveWatch(t *testing.T, w Watcher, name string) {
	t.Helper()
	if err := w.Remove(name); err != nil {
		t.Fatal(err)
	}
}

//----------

func readEvent(t *testing.T, w Watcher, failOnTimeout bool, fn func(*Event) bool) {
	t.Helper()
	tick := time.NewTicker(3000 * time.Millisecond)
	defer tick.Stop()
	select {
	case <-tick.C:
		if failOnTimeout {
			t.Fatal("event timeout")
		}
	case ev := <-w.Events():
		if err, ok := ev.(error); ok {
			t.Fatal(err)
		} else if !fn(ev.(*Event)) {
			t.Fatal(ev)
		}
	}
}
