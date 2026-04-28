package main

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/app"
	"clashtui/internal/clash"
	"clashtui/internal/config"
	"clashtui/internal/proxy"
	"clashtui/internal/singleinstance"
)

func main() {
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
		}
	}

	runTUI()
}

func printStatus() {
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

	// Keep running until killed
	select {}
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
	defer cleanupOnExit()

	// Only stop clash if API is not responding (truly leftover process)
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