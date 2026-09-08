//go:build !windows

package compiler

import (
	"errors"
	"os"
	"syscall"
)

func processAlive(pid int) (bool, error) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}
	defer process.Release()
	err = process.Signal(syscall.Signal(0))
	if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
		return false, nil
	}
	return true, err
}
