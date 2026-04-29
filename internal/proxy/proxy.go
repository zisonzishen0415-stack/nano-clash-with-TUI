package proxy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ProxyAddr = "127.0.0.1:7890"

// SetSystemProxy sets proxy in current process AND persists to ~/.pam_environment
func SetSystemProxy() error {
	// Set in current process (immediate effect)
	os.Setenv("HTTP_PROXY", "http://"+ProxyAddr)
	os.Setenv("HTTPS_PROXY", "http://"+ProxyAddr)
	os.Setenv("ALL_PROXY", "socks5://"+ProxyAddr)
	os.Setenv("NO_PROXY", "localhost,127.0.0.1")

	// Persist to ~/.pam_environment for future sessions
	return updatePamEnvironment(true)
}

// UnsetSystemProxy removes proxy from current process AND ~/.pam_environment
func UnsetSystemProxy() error {
	// Remove from current process
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")

	// Remove from ~/.pam_environment
	return updatePamEnvironment(false)
}

// updatePamEnvironment adds or removes proxy settings from ~/.pam_environment
func updatePamEnvironment(setProxy bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pamFile := filepath.Join(home, ".pam_environment")

	// Read existing content
	var lines []string
	f, err := os.Open(pamFile)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			// Skip proxy-related lines
			if !strings.HasPrefix(line, "HTTP_PROXY") &&
				!strings.HasPrefix(line, "HTTPS_PROXY") &&
				!strings.HasPrefix(line, "ALL_PROXY") &&
				!strings.HasPrefix(line, "NO_PROXY") {
				lines = append(lines, line)
			}
		}
		f.Close()
	}

	// Add proxy lines if setting
	if setProxy {
		lines = append(lines,
			fmt.Sprintf("HTTP_PROXY DEFAULT=http://%s", ProxyAddr),
			fmt.Sprintf("HTTPS_PROXY DEFAULT=http://%s", ProxyAddr),
			fmt.Sprintf("ALL_PROXY DEFAULT=socks5://%s", ProxyAddr),
			"NO_PROXY DEFAULT=localhost,127.0.0.1",
		)
	}

	// Write back
	return os.WriteFile(pamFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}