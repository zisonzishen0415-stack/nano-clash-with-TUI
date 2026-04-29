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
	"clashtui/internal/singleinstance"
)

func main() {
	args := os.Args[1:]

	// 启动时检查：如果 clash 没运行，清除代理设置（解决关机后遗留问题）
	cleanStaleProxySettings()

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
		case "--toggle":
			toggleProxy()
			return
		}
	}

	runTUI()
}

// cleanStaleProxySettings 检查 clash 是否运行，如果没运行就清除代理设置
func cleanStaleProxySettings() {
	client := clash.NewClient()
	if !client.IsConnected() {
		// clash 没运行，清除可能遗留的代理设置
		proxy.UnsetSystemProxy()
	}
}

func printStatus() {
	// 直接查询 Clash API，获取真实状态
	client := clash.NewClient()
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

	// Start clash if config exists
	if config.Exists() {
		core := clash.NewCore()
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		core.Start()
		proxy.SetSystemProxy()
	}

	// Wait for termination signal
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
	// TUI 退出时不清理 clash - 让服务继续运行

	// Only stop clash if API is not responding (truly leftover zombie process)
	client := clash.NewClient()
	if !client.IsConnected() {
		core := clash.NewCore()
		core.Stop()
	}

	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func stopAll() {
	// First, try to send SIGTERM to daemon process
	daemonPid, err := singleinstance.ReadPID()
	if err == nil && daemonPid > 0 {
		process, _ := os.FindProcess(daemonPid)
		// Send SIGTERM to daemon, it will cleanup properly via defer
		process.Signal(syscall.SIGTERM)
		// Wait for process to exit (poll since Wait() only works for children)
		for i := 0; i < 10; i++ {
			if process.Signal(syscall.Signal(0)) != nil {
				break // Process exited
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Fallback: directly stop clash (if daemon not running)
	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
	fmt.Println("Stopped")
}

func toggleProxy() {
	client := clash.NewClient()
	connected := client.IsConnected()

	if connected {
		stopAll()
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
		proxy.SetSystemProxy()
		fmt.Println("Started")
	}
}

func cleanupOnExit() {
	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
}