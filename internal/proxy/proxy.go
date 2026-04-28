package proxy

import (
	"os"
	"strings"
)

const ProxyAddr = "127.0.0.1:7890"

func SetSystemProxy() error {
	envVars := []string{
		"HTTP_PROXY=http://" + ProxyAddr,
		"HTTPS_PROXY=http://" + ProxyAddr,
		"ALL_PROXY=socks5://" + ProxyAddr,
		"NO_PROXY=localhost,127.0.0.1",
	}

	for _, v := range envVars {
		parts := strings.SplitN(v, "=", 2)
		os.Setenv(parts[0], parts[1])
	}

	home, _ := os.UserHomeDir()
	
	for _, file := range []string{home + "/.bashrc", home + "/.profile"} {
		content, _ := os.ReadFile(file)
		if !strings.Contains(string(content), "HTTP_PROXY=http://"+ProxyAddr) {
			f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\n# ClashTUI proxy\n")
				for _, v := range envVars {
					f.WriteString("export " + v + "\n")
				}
				f.Close()
			}
		}
	}

	return nil
}

func UnsetSystemProxy() error {
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")

	home, _ := os.UserHomeDir()
	for _, file := range []string{home + "/.bashrc", home + "/.profile"} {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		var newLines []string
		skip := false
		for _, line := range lines {
			if strings.Contains(line, "# ClashTUI proxy") {
				skip = true
				continue
			}
			if skip && (strings.Contains(line, "HTTP_PROXY=") ||
				strings.Contains(line, "HTTPS_PROXY=") ||
				strings.Contains(line, "ALL_PROXY=") ||
				strings.Contains(line, "NO_PROXY=") ||
				strings.Contains(line, "export no_proxy")) {
				continue
			}
			if skip && strings.TrimSpace(line) == "" {
				skip = false
			}
			newLines = append(newLines, line)
		}
		os.WriteFile(file, []byte(strings.Join(newLines, "\n")), 0644)
	}

	return nil
}