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

func getAPIPort() int {
	s := settings.Load()
	if s.APIPort == 0 {
		return 9090
	}
	return s.APIPort
}

func cleanStaleProxySettings() {
	client := clash.NewClient(getAPIPort())
	if !client.IsConnected() {
		proxy.UnsetSystemProxy()
	}
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

	if config.Exists() {
		core := clash.NewCore()
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		core.Start()
		proxy.SetSystemProxy()
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

	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
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
	fmt.Println("Stopped")
}

func toggleProxy() {
	client := clash.NewClient(getAPIPort())
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