//go:build !darwin

package cli

import "os"

type inputKey byte

const (
	keyUnknown inputKey = 0
	keyEnter   inputKey = 13
	keyCtrlC   inputKey = 3
	keyUp      inputKey = 200
	keyDown    inputKey = 201
)

func readImmediateKey(file *os.File) (inputKey, bool, error) {
	return keyUnknown, false, nil
}

func readImmediateByte(file *os.File) (byte, bool, error) {
	return 0, false, nil
}

func terminalAvailable(file *os.File) bool {
	return false
}
