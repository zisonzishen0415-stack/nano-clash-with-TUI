package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/app"
	"clashtui/internal/clash"
	"clashtui/internal/config"
	"clashtui/internal/proxy"
	"clashtui/internal/settings"
	"clashtui/internal/singleinstance"
)

func main() {
	settings.MigrateFromOldFormat()

	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "--status":
			printStatus()
			return
		case "--daemon":
			runDaemon()
			return
		case "--stop":
			stopAll()
			return
		case "--restore-network":
			restoreNetwork()
			return
		case "--toggle":
			toggleProxy()
			return
		case "--env":
			printEnv()
			return
		}
	}

	runTUI()
}

func getAPIPort() int {
	s := settings.Load()
	if s.APIPort == 0 {
		return 9090
	}
	return s.APIPort
}

func getProxyPort() int {
	s := settings.Load()
	if s.ProxyPort == 0 {
		return 7890
	}
	return s.ProxyPort
}

func printStatus() {
	// Status is read-only, no need to check for running instance
	client := clash.NewClient(getAPIPort())
	connected := client.IsConnected()

	status := map[string]string{
		"text":    "🔴",
		"tooltip": "Proxy: stopped",
		"class":   "stopped",
	}

	if connected {
		current, _ := client.GetCurrentProxy()
		status["text"] = "🟢"
		status["tooltip"] = "Proxy: " + current
		status["class"] = "running"
	}

	data, _ := json.Marshal(status)
	fmt.Println(string(data))
}

func runDaemon() {
	acquired, err := singleinstance.Acquire()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !acquired {
		fmt.Fprintln(os.Stderr, "Already running")
		os.Exit(0)
	}

	defer singleinstance.Release()
	defer cleanupOnExit()

	s := settings.Load()
	core := clash.NewCore()

	if config.Exists() {
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		core.Start()
		if s.SystemProxy {
			proxy.SetSystemProxy(s.ProxyPort)
		}
	}

	// Handle socket commands in background
	go singleinstance.HandleSocketCommands(func(cmd string) string {
		return handleCommand(cmd, core)
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

func runTUI() {
	acquired, err := singleinstance.Acquire()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !acquired {
		os.Exit(0)
	}

	defer singleinstance.Release()

	client := clash.NewClient(getAPIPort())
	core := clash.NewCore()

	// Clear stale state if mihomo not running
	if !client.IsConnected() {
		core.Stop()
	}

	// Handle socket commands in background (TUI also handles IPC)
	go singleinstance.HandleSocketCommands(func(cmd string) string {
		return handleCommand(cmd, core)
	})

	p := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Exit message
	client = clash.NewClient(getAPIPort())
	if client.IsConnected() {
		fmt.Println("\n  ✓ Exited - core running")
		fmt.Println("  Run 'clashtui --env' to see proxy env vars")
	} else {
		fmt.Println("\n  ✓ Exited - core stopped")
		fmt.Println("  Run in shell: unset HTTP_PROXY HTTPS_PROXY ALL_PROXY")
	}
}

// handleCommand handles IPC commands from socket
// This runs in the TUI/daemon process
func handleCommand(cmd string, core *clash.Core) string {
	s := settings.Load()

	switch cmd {
	case "stop":
		core.Stop()
		proxy.UnsetSystemProxy()
		return "ok: stopped"

	case "toggle":
		client := clash.NewClient(s.APIPort)
		if client.IsConnected() {
			core.Stop()
			proxy.UnsetSystemProxy()
			return "ok: stopped"
		} else {
			if !config.Exists() {
				return "error: no config"
			}
			if !core.IsInstalled() {
				core.Install()
				core.DownloadGeoData()
			}
			core.Start()
			if s.SystemProxy {
				proxy.SetSystemProxy(s.ProxyPort)
			}
			return "ok: started"
		}

	case "restore-network":
		core.Stop()
		proxy.UnsetSystemProxy()
		return "ok: network restored"

	case "status":
		client := clash.NewClient(s.APIPort)
		if client.IsConnected() {
			current, _ := client.GetCurrentProxy()
			return "running: " + current
		}
		return "stopped"

	default:
		return "error: unknown command"
	}
}

// stopAll stops mihomo and clears proxy
// If TUI/daemon is running, delegate via socket; otherwise operate directly
func stopAll() {
	// Check if TUI/daemon is running via socket
	if singleinstance.IsRunning() {
		resp, err := singleinstance.SendCommand("stop")
		if err == nil {
			fmt.Println(resp)
			fmt.Println("Terminal: source ~/.config/clashtui/proxy.sh")
			return
		}
		// Socket failed but instance might be running - send signal
		pid, err := singleinstance.ReadPID()
		if err == nil && pid > 0 {
			process, _ := os.FindProcess(pid)
			process.Signal(syscall.SIGTERM)
			time.Sleep(500 * time.Millisecond)
		}
	}

	// No running instance - operate directly
	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
	fmt.Println("Stopped, proxy cleared")
	fmt.Println("Terminal: source ~/.config/clashtui/proxy.sh")
}

// restoreNetwork forcefully clears all proxy settings
// This is an emergency command, always operate directly (even if TUI running)
func restoreNetwork() {
	// Emergency: force clear, don't wait for socket response
	// Kill TUI/daemon first if running
	pid, err := singleinstance.ReadPID()
	if err == nil && pid > 0 {
		process, _ := os.FindProcess(pid)
		process.Signal(syscall.SIGTERM)
		time.Sleep(200 * time.Millisecond)
	}

	// Kill any lingering mihomo process
	core := clash.NewCore()
	core.Stop()

	// Clear all proxy settings
	proxy.UnsetSystemProxy()

	fmt.Println("✓ Network restored!")
	fmt.Println("✓ Proxy settings cleared")
	fmt.Println("")
	fmt.Println("If DNS still broken, check symlink:")
	fmt.Println("  ls -la /etc/resolv.conf")
	fmt.Println("  Should point to: /run/systemd/resolve/resolv.conf")
	fmt.Println("  If broken, fix with:")
	fmt.Println("    sudo ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf")
	fmt.Println("")
	fmt.Println("Or restart DNS service:")
	fmt.Println("  sudo systemctl restart systemd-resolved")
}

// toggleProxy toggles proxy on/off
// If TUI/daemon is running, delegate via socket; otherwise operate directly
func toggleProxy() {
	// Check if TUI/daemon is running via socket
	if singleinstance.IsRunning() {
		resp, err := singleinstance.SendCommand("toggle")
		if err == nil {
			fmt.Println(resp)
			return
		}
	}

	// No running instance - operate directly
	client := clash.NewClient(getAPIPort())
	connected := client.IsConnected()
	s := settings.Load()

	if connected {
		core := clash.NewCore()
		core.Stop()
		proxy.UnsetSystemProxy()
		fmt.Println("Stopped")
	} else {
		if !config.Exists() {
			fmt.Println("No config, run TUI first")
			os.Exit(1)
		}
		core := clash.NewCore()
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		core.Start()
		if s.SystemProxy {
			proxy.SetSystemProxy(s.ProxyPort)
		}
		fmt.Println("Started")
	}
}

func printEnv() {
	port := getProxyPort()
	fmt.Printf("export HTTP_PROXY=http://127.0.0.1:%d\n", port)
	fmt.Printf("export HTTPS_PROXY=http://127.0.0.1:%d\n", port)
	fmt.Printf("export ALL_PROXY=socks5://127.0.0.1:%d\n", port)
}

func cleanupOnExit() {
	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
}