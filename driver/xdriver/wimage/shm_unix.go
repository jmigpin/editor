//go:build linux || (darwin && !ios)

package wimage

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func systemShmOps() shmOps {
	return shmOps{
		get: func(size int) (uintptr, error) {
			id, err := unix.SysvShmGet(unix.IPC_PRIVATE, size, 0600)
			return uintptr(id), err
		},
		attach: func(id uintptr) (uintptr, error) {
			buf, err := unix.SysvShmAttach(int(id), 0, 0)
			if err != nil {
				return 0, err
			}
			return uintptr(unsafe.Pointer(&buf[0])), nil
		},
		detach: func(addr uintptr) error {
			// SysvShmDetach only uses the slice's start address, not its length.
			return unix.SysvShmDetach(unsafe.Slice((*byte)(unsafe.Pointer(addr)), 1))
		},
		remove: func(id uintptr) error {
			_, err := unix.SysvShmCtl(int(id), unix.IPC_RMID, nil)
			return err
		},
	}
}
