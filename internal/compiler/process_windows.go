//go:build windows

package compiler

import (
	"errors"
	"syscall"
)

func processAlive(pid int) (bool, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if errors.Is(err, syscall.Errno(87)) {
		return false, nil
	} // ERROR_INVALID_PARAMETER: no such process
	if err != nil {
		return false, err
	}
	defer syscall.CloseHandle(handle)
	var code uint32
	if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
		return false, err
	}
	return code == 259, nil // STILL_ACTIVE
}
