//go:build freebsd || netbsd || dragonfly

package wimage

import "golang.org/x/sys/unix"

// These BSD targets support direct syscalls, but x/sys/unix does not expose SysvShm wrappers for them.
func systemShmOps() shmOps {
	return shmOps{
		get: func(size int) (uintptr, error) {
			id, _, errno := unix.Syscall(unix.SYS_SHMGET, 0, uintptr(size), 0600)
			return id, shmSyscallError(errno)
		},
		attach: func(id uintptr) (uintptr, error) {
			addr, _, errno := unix.Syscall(unix.SYS_SHMAT, id, 0, 0)
			return addr, shmSyscallError(errno)
		},
		detach: func(addr uintptr) error {
			_, _, errno := unix.Syscall(unix.SYS_SHMDT, addr, 0, 0)
			return shmSyscallError(errno)
		},
		remove: func(id uintptr) error {
			_, _, errno := unix.Syscall(unix.SYS_SHMCTL, id, 0, 0) // IPC_RMID is zero on these targets.
			return shmSyscallError(errno)
		},
	}
}

func shmSyscallError(errno unix.Errno) error {
	if errno != 0 {
		return errno
	}
	return nil
}
