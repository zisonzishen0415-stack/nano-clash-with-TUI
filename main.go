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

	if config.Exists() {
		core := clash.NewCore()
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		core.Start()
		if s.SystemProxy {
			proxy.SetSystemProxy(s.ProxyPort)
		}
	}

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
	if !client.IsConnected() {
		core := clash.NewCore()
		core.Stop()
	}

	p := tea.NewProgram(app.New())
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

func stopAll() {
	daemonPid, err := singleinstance.ReadPID()
	if err == nil && daemonPid > 0 {
		process, _ := os.FindProcess(daemonPid)
		process.Signal(syscall.SIGTERM)
		for i := 0; i < 10; i++ {
			if process.Signal(syscall.Signal(0)) != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	core := clash.NewCore()
	core.Stop()
	proxy.UnsetSystemProxy()
	fmt.Println("Stopped, proxy cleared")
	fmt.Println("Terminal: source ~/.config/clashtui/proxy.sh")
}

func toggleProxy() {
	client := clash.NewClient(getAPIPort())
	connected := client.IsConnected()
	s := settings.Load()

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