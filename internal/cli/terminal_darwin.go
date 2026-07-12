package cli

import (
	"os"
	"syscall"
	"unsafe"
)

type inputKey byte

const (
	keyUnknown inputKey = 0
	keyEnter   inputKey = 13
	keyCtrlC   inputKey = 3
	keyUp      inputKey = 200
	keyDown    inputKey = 201
)

func readImmediateKey(file *os.File) (inputKey, bool, error) {
	key, ok, err := readImmediateByte(file)
	if err != nil || !ok {
		return keyUnknown, ok, err
	}
	switch key {
	case '\r', '\n':
		return keyEnter, true, nil
	case 3:
		return keyCtrlC, true, nil
	case 27:
		second, ok, err := readImmediateByte(file)
		if err != nil || !ok {
			return keyUnknown, ok, err
		}
		third, ok, err := readImmediateByte(file)
		if err != nil || !ok {
			return keyUnknown, ok, err
		}
		if second == '[' {
			switch third {
			case 'A':
				return keyUp, true, nil
			case 'B':
				return keyDown, true, nil
			}
		}
		return keyUnknown, true, nil
	default:
		return inputKey(key), true, nil
	}
}

func readImmediateByte(file *os.File) (byte, bool, error) {
	fd := file.Fd()
	original, err := getTermios(fd)
	if err != nil {
		return 0, false, nil
	}

	raw := original
	raw.Lflag &^= syscall.ICANON | syscall.ECHO
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if err := setTermios(fd, raw); err != nil {
		return 0, false, err
	}
	defer func() {
		_ = setTermios(fd, original)
	}()

	var buf [1]byte
	n, err := syscall.Read(int(fd), buf[:])
	if err != nil {
		return 0, true, err
	}
	if n == 0 {
		return 0, true, nil
	}
	return buf[0], true, nil
}

func terminalAvailable(file *os.File) bool {
	_, err := getTermios(file.Fd())
	return err == nil
}

func getTermios(fd uintptr) (syscall.Termios, error) {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&termios)))
	if errno != 0 {
		return termios, errno
	}
	return termios, nil
}

func setTermios(fd uintptr, termios syscall.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(&termios)))
	if errno != 0 {
		return errno
	}
	return nil
}
