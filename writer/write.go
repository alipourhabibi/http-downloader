package writer

import (
	"sync"
	"syscall"
)

type file struct {
	sync.Mutex
	fd int
}

func NewFile(f string) *file {
	fd, err := syscall.Open(f, syscall.O_RDWR, 0644)
	if err != nil {
		//TODO
	}
	return &file{
		fd: fd,
	}
}

func Create(file string) error {
	_, err := syscall.Creat(file, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (f *file) Save(data []byte, offset int64) (int, error) {
	return syscall.Pwrite(f.fd, data, offset)
}

func (f *file) Close() {
	syscall.Close(f.fd)
}
