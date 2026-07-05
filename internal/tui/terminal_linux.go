//go:build linux

package tui

import (
	"syscall"
	"unsafe"
)

const (
	ioctlTCGETS  = 0x5401
	ioctlTCSETS  = 0x5402
	ioctlTIOCGWZ = 0x5413
)

type terminalSize struct {
	Rows uint16
	Cols uint16
	X    uint16
	Y    uint16
}

func enableRawMode(fd int) (func(), error) {
	var original syscall.Termios
	if err := ioctl(fd, ioctlTCGETS, uintptr(unsafe.Pointer(&original))); err != nil {
		return nil, err
	}

	raw := original
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 0
	raw.Cc[syscall.VTIME] = 0

	if err := ioctl(fd, ioctlTCSETS, uintptr(unsafe.Pointer(&raw))); err != nil {
		return nil, err
	}

	return func() {
		_ = ioctl(fd, ioctlTCSETS, uintptr(unsafe.Pointer(&original)))
	}, nil
}

func terminalDimensions(fd int) (int, int) {
	var size terminalSize
	if err := ioctl(fd, ioctlTIOCGWZ, uintptr(unsafe.Pointer(&size))); err != nil {
		return 80, 24
	}
	if size.Cols == 0 || size.Rows == 0 {
		return 80, 24
	}
	return int(size.Cols), int(size.Rows)
}

func readAvailable(fd int, buf []byte) (int, error) {
	n, err := syscall.Read(fd, buf)
	if err == syscall.EINTR || err == syscall.EAGAIN {
		return 0, nil
	}
	return n, err
}

func ioctl(fd int, request uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
