package fileswatcher

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"testing"
	"time"
)

func tempDir() string {
	name, err := ioutil.TempDir("", "wtest")
	if err != nil {
		panic(err)
	}
	return name
}
func mkDir(t *testing.T, name string) {
	err := os.MkdirAll(name, 0755)
	if err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, name string) {
	//err := ioutil.WriteFile(name, []byte("ABC"), 0644)
	//if err != nil {
	//	t.Fatal(err)
	//}

	f, err := os.OpenFile(name, os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("ABC")
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err2 := f.Close(); err2 != nil {
		err = err2
	}
	if err != nil {
		t.Fatal(err)
	}
}
func createFile(t *testing.T, name string) {
	//err := ioutil.WriteFile(name, []byte{}, 0644)
	//if err != nil {
	//	t.Fatal(err)
	//}

	f, err := os.OpenFile(name, os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func newTargetWatcherForTest(t *testing.T) *TargetWatcher {
	var mu sync.Mutex
	logf := func(f string, a ...interface{}) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		t.Logf(f, a...)
	}

	w, err := NewTargetWatcher(logf)
	if err != nil {
		t.Fatal(err)
	}

	return w
}

//func sleepToGetEvents() {
//	time.Sleep(2000 * time.Millisecond)
//}

func waitForTEvent(t *testing.T, w *TargetWatcher, fn func(interface{}) bool) {
	t.Helper()
	waitForEvent(t, w.Events, true, fn)
}

func waitForPossibleTEvent(t *testing.T, w *TargetWatcher, fn func(interface{}) bool) {
	waitForEvent(t, w.Events, false, fn)
}

func testWatchLeaks(t *testing.T, w *TargetWatcher) {
	// allow late events to be handled
	time.Sleep(50 * time.Millisecond)

	l1 := len(w.entries.m)
	l2 := len(w.w.watches.m)

	if l1 != l2 {
		t.Logf("watch entries: %v %v", l1, l2)

		t.Logf("watching %d", len(w.entries.m))
		for k := range w.entries.m {
			w.Logf("k=%v", k)
		}
		t.Logf("inner watching %d", len(w.w.watches.m))
		for k := range w.w.watches.m {
			w.Logf("k=%v", k)
		}

		t.Fatalf("leaking watch entries")
	}

}

//----------------------------------------------------------

func TestTargetWatcher1(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	file1 := path.Join(dir, "file1.txt")

	w.Add(file1)
	t.Logf("creating file1")
	createFile(t, file1)
	t.Logf("creating file1 done")

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher2(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	file1 := path.Join(dir2, "file1.txt")

	mkDir(t, dir2)
	w.Add(file1)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher3(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	w.Add(file1)
	mkDir(t, dir4)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher4(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mkDir(t, dir4)
	createFile(t, file1)
	w.Add(file1)
	os.RemoveAll(dir2)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasDelete()
	})

	mkDir(t, dir4)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher5(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mkDir(t, dir4)
	w.Add(file1)
	os.RemoveAll(dir2)
	mkDir(t, dir4)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher6(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	mkDir(t, dir4)
	w.Add(file1)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	writeFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasModify()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher7(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")
	//file2 := path.Join(dir4, "file2.txt")

	w.Add(file1)
	mkDir(t, dir4)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	os.RemoveAll(dir2) // delete

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasDelete()
	})

	for i := 0; i < 10; i++ {
		mkDir(t, dir2)
		os.RemoveAll(dir2)
		mkDir(t, dir3)
		os.RemoveAll(dir3)
		mkDir(t, dir4)
		os.RemoveAll(dir4)
		mkDir(t, dir4)
		os.RemoveAll(dir2)
		mkDir(t, dir2)
		os.RemoveAll(dir2)
		mkDir(t, dir4)
	}

	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher8(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")
	file2 := path.Join(dir4, "file2.txt")

	mkDir(t, dir4)
	createFile(t, file1)
	w.Add(file1)
	w.Add(file2)
	w.Add(dir4)
	w.Remove(file1)
	w.Remove(file2)
	createFile(t, file2)
	os.RemoveAll(dir4)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Filename == file2 && ev2.Op.HasCreate()
	})

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == dir4 && ev2.Op.HasDelete()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher9(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	dir2 := path.Join(dir, "dir2")
	dir3 := path.Join(dir2, "dir3")
	dir4 := path.Join(dir3, "dir4")
	file1 := path.Join(dir4, "file1.txt")

	w.Add(file1)
	mkDir(t, dir4)
	createFile(t, file1)

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasCreate()
	})

	testWatchLeaks(t, w)
}

func TestTargetWatcher10(t *testing.T) {
	tmpDir := tempDir()
	defer os.RemoveAll(tmpDir)
	w := newTargetWatcherForTest(t)
	defer w.Close()

	dir := tmpDir
	file1 := path.Join(dir, "file1.txt")
	file2 := path.Join(dir, "file2.txt")

	createFile(t, file1)
	w.Add(file1)

	err := os.Rename(file1, file2)
	if err != nil {
		t.Fatal(err)
	}

	waitForTEvent(t, w, func(ev interface{}) bool {
		ev2 := ev.(*Event)
		return ev2.Name == file1 && ev2.Op.HasDelete()
	})

	testWatchLeaks(t, w)
}
