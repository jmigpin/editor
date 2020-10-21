// +build !windows

package wimage

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// These constants are from /usr/include/linux/ipc.h
const (
	ipcPrivate = 0
	ipcRmID    = 0
)

func ShmOpen(size int) (shmid, addr uintptr, err error) {
	shmid, _, errno0 := unix.Syscall(unix.SYS_SHMGET, ipcPrivate, uintptr(size), 0600)
	if errno0 != 0 {
		return 0, 0, fmt.Errorf("shmget: %v", errno0)
	}
	p, _, errno1 := unix.Syscall(unix.SYS_SHMAT, shmid, 0, 0)
	if errno1 != 0 {
		return 0, 0, fmt.Errorf("shmat: %v", errno1)
	}
	return shmid, p, nil
}

func ShmClose(shmid, addr uintptr) error {
	_, _, errno := unix.Syscall(unix.SYS_SHMDT, addr, 0, 0)
	_, _, errno2 := unix.Syscall(unix.SYS_SHMCTL, shmid, ipcRmID, 0)
	if errno != 0 {
		return fmt.Errorf("shmdt: %v", errno)
	}
	if errno2 != 0 {
		return fmt.Errorf("shmctl: %v", errno2)
	}
	return nil
}
