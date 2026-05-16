package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"clashtui/internal/app"
	"clashtui/internal/clash"
	"clashtui/internal/config"
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
		case "--test-download":
			testDownload()
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
		fmt.Fprintln(os.Stderr, "Error acquiring lock:", err)
		fmt.Fprintln(os.Stderr, "  Suggestion: Check /tmp/clashtui.pid or run --restore-network")
		os.Exit(1)
	}
	if !acquired {
		fmt.Fprintln(os.Stderr, "Already running")
		os.Exit(0)
	}

	defer singleinstance.Release()
	defer cleanupOnExit()

	core := clash.NewCore()

	if config.Exists() {
		if !core.IsInstalled() {
			if err := core.Install(); err != nil {
				fmt.Fprintln(os.Stderr, "Error installing core:", err)
				fmt.Fprintln(os.Stderr, "  Suggestion: Check network and try again")
				os.Exit(1)
			}
			core.DownloadGeoData()
		}

		// Process config for TUN mode if enabled
		s := settings.Load()
		if s.TUNMode {
			data, err := config.LoadConfigNoValidation()
			if err == nil {
				newData := clash.ProcessConfigForTUN(data, true)
				config.SaveConfig(newData)
			}
		}

		if err := core.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "Error starting core:", err)
			fmt.Fprintln(os.Stderr, "  Suggestion: Run 'clashtui --restore-network' to fix network")
			os.Exit(1)
		}
	}

	go singleinstance.HandleSocketCommands(func(cmd string) string {
		return handleCommand(cmd, core)
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

func runTUI() {
	if singleinstance.IsRunning() {
		fmt.Println("Daemon already running, connecting...")
	} else {
		fmt.Println("Daemon not running, starting...")
		cmd := exec.Command(os.Args[0], "--daemon")
		cmd.Start()
		time.Sleep(2 * time.Second)
	}

	p := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		fmt.Fprintln(os.Stderr, "  Suggestion: Check terminal compatibility")
		os.Exit(1)
	}

	client := clash.NewClient(getAPIPort())
	if client.IsConnected() {
		fmt.Println("\n  ✓ Exited - core running (TUN mode)")
	} else {
		fmt.Println("\n  ✓ Exited - core stopped")
	}
}

func handleCommand(cmd string, core *clash.Core) string {
	s := settings.Load()

	switch cmd {
	case "stop":
		core.Stop()
		return "ok: stopped"

	case "toggle":
		client := clash.NewClient(s.APIPort)
		if client.IsConnected() {
			core.Stop()
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
			return "ok: started"
		}

	case "restore-network":
		core.Stop()
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

func stopAll() {
	if singleinstance.IsRunning() {
		resp, err := singleinstance.SendCommand("stop")
		if err == nil {
			fmt.Println(resp)
			return
		}
		pid, err := singleinstance.ReadPID()
		if err == nil && pid > 0 {
			process, _ := os.FindProcess(pid)
			process.Signal(syscall.SIGTERM)
			time.Sleep(500 * time.Millisecond)
		}
	}

	core := clash.NewCore()
	core.Stop()
	fmt.Println("✓ Stopped")
}

func restoreNetwork() {
	pid, err := singleinstance.ReadPID()
	if err == nil && pid > 0 {
		process, _ := os.FindProcess(pid)
		process.Signal(syscall.SIGTERM)
		time.Sleep(200 * time.Millisecond)
	}

	core := clash.NewCore()
	core.Stop()

	fmt.Println("✓ Network restored!")
	fmt.Println("✓ Core stopped")
	fmt.Println("")
	fmt.Println("If DNS still broken:")
	fmt.Println("  sudo systemctl restart systemd-resolved")
	fmt.Println("")
	fmt.Println("Alternative recovery:")
	fmt.Println("  Reboot system if network issues persist")
}

func toggleProxy() {
	if singleinstance.IsRunning() {
		resp, err := singleinstance.SendCommand("toggle")
		if err == nil {
			fmt.Println(resp)
			return
		}
	}

	client := clash.NewClient(getAPIPort())
	connected := client.IsConnected()

	if connected {
		core := clash.NewCore()
		core.Stop()
		fmt.Println("Stopped")
	} else {
		if !config.Exists() {
			fmt.Println("No config, run TUI first")
			os.Exit(1)
		}
		s := settings.Load()
		core := clash.NewCore()
		if !core.IsInstalled() {
			core.Install()
			core.DownloadGeoData()
		}
		// Process config for TUN mode if enabled
		if s.TUNMode {
			data, err := config.LoadConfigNoValidation()
			if err == nil {
				newData := clash.ProcessConfigForTUN(data, true)
				config.SaveConfig(newData)
			}
		}
		core.Start()
		fmt.Println("Started")
	}
}

func printEnv() {
	port := getProxyPort()
	fmt.Printf("export HTTP_PROXY=http://127.0.0.1:%d\n", port)
	fmt.Printf("export HTTPS_PROXY=http://127.0.0.1:%d\n", port)
	fmt.Printf("export ALL_PROXY=socks5://127.0.0.1:%d\n", port)
}

func testDownload() {
	s := settings.Load()
	sub := settings.GetActiveSubscription(s)
	if sub == nil {
		fmt.Println("❌ No active subscription")
		return
	}
	fmt.Println("Downloading:", sub.URL)

	_, info, err := clash.DownloadSubscription(sub.URL, s.ProxyPort, s.APIPort, s.TUNMode)
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}

	fmt.Println("✅ Download OK")
	fmt.Println("Traffic:", info.Traffic)
	fmt.Println("Expiry:", info.Expiry)

	if config.Exists() {
		data, _ := config.LoadConfig()
		fmt.Println("Config size:", len(data), "bytes")
	}

	core := clash.NewCore()
	if err := core.DownloadGeoData(); err != nil {
		fmt.Println("❌ GeoData error:", err)
		return
	}
	fmt.Println("✅ GeoData extracted")

	if err := core.StartAndCheck(); err != nil {
		fmt.Println("❌ Core error:", err)
		return
	}
	fmt.Println("✅ Core running")
	core.Stop()
}

func cleanupOnExit() {
	core := clash.NewCore()
	core.Stop()
}
