//go:build openbsd && (amd64 || arm64)

package wimage

import (
	"syscall"
	_ "unsafe"
)

// OpenBSD requires libc entry points; use the same runtime bridges as x/sys/unix without enabling CGo.
func systemShmOps() shmOps {
	return shmOps{
		get: func(size int) (uintptr, error) {
			id, _, errno := shmLibcCall(shmGetAddr, 0, uintptr(size), 0600)
			return id, shmLibcError(errno)
		},
		attach: func(id uintptr) (uintptr, error) {
			// The pointer-returning bridge checks the full-width return value on 64-bit targets.
			addr, _, errno := shmLibcCallPointer(shmAtAddr, id, 0, 0)
			return addr, shmLibcError(errno)
		},
		detach: func(addr uintptr) error {
			_, _, errno := shmLibcCall(shmDtAddr, addr, 0, 0)
			return shmLibcError(errno)
		},
		remove: func(id uintptr) error {
			_, _, errno := shmLibcCall(shmCtlAddr, id, 0, 0) // IPC_RMID is zero on OpenBSD.
			return shmLibcError(errno)
		},
	}
}

func shmLibcError(errno syscall.Errno) error {
	if errno != 0 {
		return errno
	}
	return nil
}

//go:linkname shmLibcCall syscall.syscall
func shmLibcCall(fn, a1, a2, a3 uintptr) (uintptr, uintptr, syscall.Errno)

//go:linkname shmLibcCallPointer syscall.syscallX
func shmLibcCallPointer(fn, a1, a2, a3 uintptr) (uintptr, uintptr, syscall.Errno)

var shmGetAddr, shmAtAddr, shmDtAddr, shmCtlAddr uintptr

//go:cgo_import_dynamic libc_wimage_shmget shmget "libc.so"
//go:cgo_import_dynamic libc_wimage_shmat shmat "libc.so"
//go:cgo_import_dynamic libc_wimage_shmdt shmdt "libc.so"
//go:cgo_import_dynamic libc_wimage_shmctl shmctl "libc.so"
