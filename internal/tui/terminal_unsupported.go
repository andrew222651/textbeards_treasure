//go:build !linux && !darwin

package tui

import "errors"

func enableRawMode(int) (func(), error) {
	return nil, errors.New("raw terminal mode is only implemented for linux and darwin")
}

func terminalDimensions(int) (int, int) {
	return 80, 24
}

func readAvailable(int, []byte) (int, error) {
	return 0, errors.New("terminal input is only implemented for linux and darwin")
}
