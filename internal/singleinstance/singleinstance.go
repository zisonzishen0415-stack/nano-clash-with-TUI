package singleinstance

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
)

const pidFile = "/tmp/clashTUI.pid"

func Acquire() (bool, error) {
	pid, err := readPID()
	if err != nil {
		if os.IsNotExist(err) {
			return writePID()
		}
		return false, err
	}

	if isRunning(pid) {
		if err := syscall.Kill(pid, syscall.SIGUSR1); err == nil {
			return false, errors.New("instance already running, sent wake signal")
		}
	}

	return writePID()
}

func Release() error {
	return os.Remove(pidFile)
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func writePID() (bool, error) {
	pid := os.Getpid()
	err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
	return true, err
}

func isRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}