//go:build !linux && (!darwin || ios) && !freebsd && !netbsd && !dragonfly && !(openbsd && (amd64 || arm64))

package wimage

import (
	"errors"
	"fmt"
	"runtime"
)

func systemShmOps() shmOps {
	err := fmt.Errorf("shared memory on %s/%s: %w", runtime.GOOS, runtime.GOARCH, errors.ErrUnsupported)
	return shmOps{
		get:    func(int) (uintptr, error) { return 0, err },
		attach: func(uintptr) (uintptr, error) { return 0, err },
		detach: func(uintptr) error { return err },
		remove: func(uintptr) error { return err },
	}
}
