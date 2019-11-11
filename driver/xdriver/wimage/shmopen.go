// +build !windows

package wimage

import (
	"fmt"
	"syscall"
)

// These constants are from /usr/include/linux/ipc.h
const (
	ipcPrivate = 0
	ipcRmID    = 0
)

func ShmOpen(size int) (shmid, addr uintptr, err error) {
	shmid, _, errno0 := syscall.RawSyscall(syscall.SYS_SHMGET, ipcPrivate, uintptr(size), 0600)
	if errno0 != 0 {
		return 0, 0, fmt.Errorf("shmget: %v", errno0)
	}
	p, _, errno1 := syscall.RawSyscall(syscall.SYS_SHMAT, shmid, 0, 0)
	if errno1 != 0 {
		return 0, 0, fmt.Errorf("shmat: %v", errno1)
	}
	return shmid, p, nil
}

func ShmClose(shmid, addr uintptr) error {
	_, _, errno := syscall.RawSyscall(syscall.SYS_SHMDT, addr, 0, 0)
	_, _, errno2 := syscall.RawSyscall(syscall.SYS_SHMCTL, shmid, ipcRmID, 0)
	if errno != 0 {
		return fmt.Errorf("shmdt: %v", errno)
	}
	if errno2 != 0 {
		return fmt.Errorf("shmctl: %v", errno2)
	}
	return nil
}
