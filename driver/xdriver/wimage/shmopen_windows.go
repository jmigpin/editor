package wimage

import "errors"

func ShmOpen(size int) (shmid, addr uintptr, err error) {
	return 0, 0, errors.New("todo: shm windows support")
}

func ShmClose(shmid, addr uintptr) error {
	return errors.New("todo: shm windows support")
}
