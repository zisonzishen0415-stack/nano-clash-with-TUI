package singleinstance

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const pidFile = "/tmp/clashtui.pid"
const socketPath = "/tmp/clashtui.sock"

var (
	lockFile *os.File
	socket   net.Listener
	mu       sync.Mutex
)

// Acquire acquires exclusive lock using flock (atomic kernel-level lock)
// Returns true if acquired, false if another instance is running
func Acquire() (bool, error) {
	mu.Lock()
	defer mu.Unlock()

	// Open/create lock file
	f, err := os.OpenFile(pidFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false, err
	}

	// Try exclusive flock (non-blocking)
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			// Another instance holds the lock
			return false, nil
		}
		return false, err
	}

	// We have the lock - write our PID
	pid := os.Getpid()
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(fmt.Sprintf("%d", pid))

	lockFile = f

	// Create Unix socket for IPC
	if err := createSocket(); err != nil {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		lockFile = nil
		return false, err
	}

	return true, nil
}

// Release releases the lock and closes socket
func Release() error {
	mu.Lock()
	defer mu.Unlock()

	if socket != nil {
		socket.Close()
		socket = nil
		os.Remove(socketPath)
	}

	if lockFile != nil {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		lockFile = nil
		os.Remove(pidFile)
	}

	return nil
}

// ReadPID reads the PID of the running instance (if any)
// This does NOT acquire lock, just reads the file
func ReadPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// IsRunning checks if another instance is running by trying to connect to its socket
func IsRunning() bool {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// SendCommand sends a command to the running instance via Unix socket
// Returns response from the running instance
func SendCommand(cmd string) (string, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return "", errors.New("no running instance")
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	conn.Write([]byte(cmd + "\n"))

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// createSocket creates Unix domain socket for IPC
func createSocket() error {
	// Remove stale socket if exists
	os.Remove(socketPath)

	s, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	// Set socket permissions (allow same user only)
	os.Chmod(socketPath, 0600)

	socket = s
	return nil
}

// GetSocket returns the socket listener for handling commands
func GetSocket() net.Listener {
	mu.Lock()
	defer mu.Unlock()
	return socket
}

// HandleSocketCommands handles incoming commands on the socket
// This should be called in a goroutine while TUI/daemon is running
func HandleSocketCommands(handler func(cmd string) string) {
	s := GetSocket()
	if s == nil {
		return
	}

	for {
		conn, err := s.Accept()
		if err != nil {
			return // Socket closed
		}

		go func(c net.Conn) {
			defer c.Close()
			c.SetDeadline(time.Now().Add(5 * time.Second))

			buf := make([]byte, 256)
			n, err := c.Read(buf)
			if err != nil {
				return
			}

			cmd := string(buf[:n])
			cmd = cmd[:len(cmd)-1] // Remove newline

			response := handler(cmd)
			c.Write([]byte(response))
		}(conn)
	}
}