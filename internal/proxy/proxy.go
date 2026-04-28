package proxy

import (
	"os"
	"os/exec"
	"strings"
)

const ProxyAddr = "127.0.0.1:7890"

func SetSystemProxy() error {
	// 设置环境变量（写入 ~/.bashrc 和 ~/.profile）
	envVars := []string{
		"HTTP_PROXY=" + ProxyAddr,
		"HTTPS_PROXY=" + ProxyAddr,
		"ALL_PROXY=socks5://" + ProxyAddr,
		"NO_PROXY=localhost,127.0.0.1",
	}

	// 设置当前进程环境变量
	for _, v := range envVars {
		os.Setenv(v[:strings.Index(v, "=")], v[strings.Index(v, "=")+1:])
	}

	// 写入 ~/.bashrc
	home, _ := os.UserHomeDir()
	bashrc := home + "/.bashrc"

	content, _ := os.ReadFile(bashrc)
	if !strings.Contains(string(content), "HTTP_PROXY=") {
		f, err := os.OpenFile(bashrc, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString("\n# ClashTUI proxy settings\n")
			for _, v := range envVars {
				f.WriteString("export " + v + "\n")
			}
			f.WriteString("export no_proxy=\"$NO_PROXY\"\n")
			f.Close()
		}
	}

	// GNOME 系统代理（如果存在）
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", "127.0.0.1").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", "7890").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", "127.0.0.1").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", "7890").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", "127.0.0.1").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", "7890").Run()

	return nil
}

func UnsetSystemProxy() error {
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("no_proxy")

	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()

	// 清理 ~/.bashrc 中的代理设置
	home, _ := os.UserHomeDir()
	bashrc := home + "/.bashrc"

	content, err := os.ReadFile(bashrc)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, "# ClashTUI proxy settings") {
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
		if skip && line == "" {
			skip = false
			continue
		}
		newLines = append(newLines, line)
	}

	os.WriteFile(bashrc, []byte(strings.Join(newLines, "\n")), 0644)
	return nil
}