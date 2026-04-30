package proxy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SetSystemProxy sets system proxy for GUI apps and creates shell script
func SetSystemProxy(port int) {
	portStr := fmt.Sprintf("%d", port)

	// GNOME (gsettings) - browsers read this
	if hasGsettings() {
		exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", "127.0.0.1").Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", portStr).Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", "127.0.0.1").Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", portStr).Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", "127.0.0.1").Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", portStr).Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy", "ignore-hosts",
			"['localhost', '127.0.0.0/8', '::1']").Run()
	}

	// KDE
	if hasKwriteconfig() {
		kwriteCmd := getKwriteconfigCmd()
		exec.Command(kwriteCmd, "--file", "kioslaverc", "--group", "Proxy Settings",
			"--key", "ProxyType", "1").Run()
		exec.Command(kwriteCmd, "--file", "kioslaverc", "--group", "Proxy Settings",
			"--key", "httpProxy", fmt.Sprintf("http://127.0.0.1:%d", port)).Run()
		exec.Command(kwriteCmd, "--file", "kioslaverc", "--group", "Proxy Settings",
			"--key", "httpsProxy", fmt.Sprintf("http://127.0.0.1:%d", port)).Run()
		exec.Command(kwriteCmd, "--file", "kioslaverc", "--group", "Proxy Settings",
			"--key", "NoProxyFor", "localhost,127.0.0.1").Run()
	}

	// Create shell script for terminal users to source
	createProxyScript(port, true)
}

// UnsetSystemProxy clears system proxy and creates clear script
func UnsetSystemProxy() {
	// Clear GNOME
	if hasGsettings() {
		exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
	}

	// Clear KDE
	if hasKwriteconfig() {
		kwriteCmd := getKwriteconfigCmd()
		exec.Command(kwriteCmd, "--file", "kioslaverc", "--group", "Proxy Settings",
			"--key", "ProxyType", "0").Run()
	}

	// Flush DNS cache
	exec.Command("systemd-resolve", "--flush-caches").Run()

	// Create shell script to clear env vars
	createProxyScript(0, false)
}

// createProxyScript creates ~/.config/clashtui/proxy.sh for user to source
func createProxyScript(port int, enable bool) {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "clashtui")
	os.MkdirAll(configDir, 0755)

	scriptPath := filepath.Join(configDir, "proxy.sh")

	var content string
	if enable {
		content = fmt.Sprintf(`#!/bin/sh
# ClashTUI proxy env vars - run: source ~/.config/clashtui/proxy.sh
export HTTP_PROXY=http://127.0.0.1:%d
export HTTPS_PROXY=http://127.0.0.1:%d
export ALL_PROXY=socks5://127.0.0.1:%d
export NO_PROXY=localhost,127.0.0.1
echo "Proxy enabled: 127.0.0.1:%d"
`, port, port, port, port)
	} else {
		content = `#!/bin/sh
# ClashTUI proxy cleared - run: source ~/.config/clashtui/proxy.sh
unset HTTP_PROXY
unset HTTPS_PROXY
unset ALL_PROXY
unset NO_PROXY
unset no_proxy
echo "Proxy disabled"
`
	}

	os.WriteFile(scriptPath, []byte(content), 0755)
}

func hasGsettings() bool {
	_, err := exec.LookPath("gsettings")
	return err == nil
}

func hasKwriteconfig() bool {
	_, err := exec.LookPath("kwriteconfig6")
	if err == nil {
		return true
	}
	_, err = exec.LookPath("kwriteconfig5")
	return err == nil
}

func getKwriteconfigCmd() string {
	if _, err := exec.LookPath("kwriteconfig6"); err == nil {
		return "kwriteconfig6"
	}
	return "kwriteconfig5"
}