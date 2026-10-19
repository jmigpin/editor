package wimage

import (
	"errors"
	"flag"
	"image"
	"image/color"
	"reflect"
	"testing"
)

var testSystemShm = flag.Bool("wimage-shm", false, "test real shared memory allocation, mapping and release without an X server")

func TestShmOpenCleanup(t *testing.T) {
	getErr, attachErr, removeErr := errors.New("get failed"), errors.New("attach failed"), errors.New("remove failed")
	for _, tc := range []struct {
		name                         string
		getErr, attachErr, removeErr error
		calls                        []string
	}{
		{"get", getErr, nil, nil, []string{"get"}},
		{"attach", nil, attachErr, nil, []string{"get", "attach", "remove"}},
		{"attachAndRemove", nil, attachErr, removeErr, []string{"get", "attach", "remove"}},
		{"success", nil, nil, nil, []string{"get", "attach"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := []string{}
			ops := shmOps{
				get: func(size int) (uintptr, error) {
					calls = append(calls, "get")
					if size != 4096 {
						t.Fatalf("size: %d", size)
					}
					return 0, tc.getErr // Zero is a valid segment ID.
				},
				attach: func(id uintptr) (uintptr, error) {
					calls = append(calls, "attach")
					if id != 0 {
						t.Fatalf("attach ID: %d", id)
					}
					return 123, tc.attachErr
				},
				remove: func(id uintptr) error {
					calls = append(calls, "remove")
					if id != 0 {
						t.Fatalf("remove ID: %d", id)
					}
					return tc.removeErr
				},
			}
			id, addr, err := shmOpen(4096, ops)
			wantErr := tc.getErr != nil || tc.attachErr != nil
			if (err != nil) != wantErr {
				t.Fatalf("got error %v, want failure=%v", err, wantErr)
			}
			for _, want := range []error{tc.getErr, tc.attachErr, tc.removeErr} {
				if want != nil && !errors.Is(err, want) {
					t.Errorf("missing error %v in %v", want, err)
				}
			}
			if err == nil && addr != 123 {
				t.Errorf("address: %d", addr)
			}
			if err != nil && (id != 0 || addr != 0) {
				t.Errorf("failed open returned %d, %d", id, addr)
			}
			if !reflect.DeepEqual(calls, tc.calls) {
				t.Errorf("calls: got %v, want %v", calls, tc.calls)
			}
		})
	}
	if _, _, err := shmOpen(0, shmOps{}); err == nil {
		t.Fatal("zero size must fail before calling the backend")
	}
}

func TestShmCloseCleanup(t *testing.T) {
	detachErr, removeErr := errors.New("detach failed"), errors.New("remove failed")
	for _, failDetach := range []bool{false, true} {
		for _, failRemove := range []bool{false, true} {
			calls := []string{}
			err := shmClose(7, 123, shmOps{
				detach: func(addr uintptr) error {
					calls = append(calls, "detach")
					if addr != 123 {
						t.Fatalf("address: %d", addr)
					}
					if failDetach {
						return detachErr
					}
					return nil
				},
				remove: func(id uintptr) error {
					calls = append(calls, "remove")
					if id != 7 {
						t.Fatalf("ID: %d", id)
					}
					if failRemove {
						return removeErr
					}
					return nil
				},
			})
			if errors.Is(err, detachErr) != failDetach || errors.Is(err, removeErr) != failRemove {
				t.Errorf("detach=%v remove=%v: %v", failDetach, failRemove, err)
			}
			if !reflect.DeepEqual(calls, []string{"detach", "remove"}) {
				t.Errorf("calls: %v", calls)
			}
		}
	}
}

// Run natively on each target with go test ./driver/xdriver/wimage -run '^TestShmLifecycle$' -count=1 -wimage-shm.
func TestShmLifecycle(t *testing.T) {
	if !*testSystemShm {
		t.Skip("use -wimage-shm to test the system backend")
	}
	wi, err := NewShmImgWrap(image.Rect(0, 0, 32, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := wi.Close(); err != nil {
			t.Error(err)
		}
	})
	want := color.RGBA{R: 123, G: 45, B: 67, A: 255}
	wi.Img.SetRGBA(2, 3, want)
	if got := wi.Img.At(2, 3); got != want {
		t.Fatalf("pixel: got %v, want %v", got, want)
	}
	if err := wi.Close(); err != nil {
		t.Fatal(err)
	}
	if wi.Img != nil {
		t.Fatal("closed image remains drawable")
	}
	if err := wi.Close(); err != nil {
		t.Fatal(err)
	}
}
