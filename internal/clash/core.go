package clash

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"clashtui/internal/config"
)

const coreVersion = "v1.18.10"
const mmdbDownloadURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
const geositeDownloadURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"

func getCoreDownloadURL() string {
	arch := runtime.GOARCH
	if arch == "arm64" || arch == "arm" {
		arch = "arm64"
	} else {
		arch = "amd64"
	}
	return fmt.Sprintf("https://gh-proxy.com/https://github.com/MetaCubeX/mihomo/releases/download/%s/mihomo-linux-%s-%s.gz", coreVersion, arch, coreVersion)
}

type SubscriptionInfo struct {
	Traffic string
	Expiry  string
}

func parseSubscriptionInfo(urlStr string, headers http.Header) SubscriptionInfo {
	info := SubscriptionInfo{}

	u, err := url.Parse(urlStr)
	if err == nil && u.Fragment != "" {
		fragment := u.Fragment

		if strings.Contains(fragment, "流量") {
			parts := strings.Split(fragment, "|")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "流量:") || strings.HasPrefix(p, "流量=") {
					info.Traffic = strings.TrimPrefix(strings.TrimPrefix(p, "流量:"), "流量=")
				}
				if strings.Contains(p, "过期") || strings.Contains(p, "到期") {
					info.Expiry = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(p, "过期:"), "过期="), "到期:")
					info.Expiry = strings.TrimPrefix(info.Expiry, "到期=")
				}
			}
		}

		if strings.Contains(fragment, "traffic") {
			params := strings.Split(fragment, "&")
			for _, p := range params {
				if strings.HasPrefix(p, "traffic=") {
					info.Traffic = strings.TrimPrefix(p, "traffic=")
				}
				if strings.HasPrefix(p, "expire=") || strings.HasPrefix(p, "expiry=") {
					info.Expiry = strings.TrimPrefix(strings.TrimPrefix(p, "expire="), "expiry=")
				}
			}
		}
	}

	userInfo := headers.Get("subscription-userinfo")
	if userInfo != "" {
		var upload, download, total int64
		var expire int64

		parts := strings.Split(userInfo, ";")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "upload=") {
				upload, _ = strconv.ParseInt(strings.TrimPrefix(p, "upload="), 10, 64)
			}
			if strings.HasPrefix(p, "download=") {
				download, _ = strconv.ParseInt(strings.TrimPrefix(p, "download="), 10, 64)
			}
			if strings.HasPrefix(p, "total=") {
				total, _ = strconv.ParseInt(strings.TrimPrefix(p, "total="), 10, 64)
			}
			if strings.HasPrefix(p, "expire=") {
				expire, _ = strconv.ParseInt(strings.TrimPrefix(p, "expire="), 10, 64)
			}
		}

		if total > 0 {
			used := upload + download
			usedGB := used / 1024 / 1024 / 1024
			totalGB := total / 1024 / 1024 / 1024
			info.Traffic = fmt.Sprintf("%dGB/%dGB", usedGB, totalGB)
		}

		if expire > 0 {
			info.Expiry = time.Unix(expire, 0).Format("2006-01-02")
		}
	}

	return info
}

type Core struct{}

// Global process tracking to prevent zombies
var runningCmd *exec.Cmd

const clashPidFile = "clash.pid"

func NewCore() *Core { return &Core{} }

func getClashPidFilePath() string {
	return filepath.Join(config.GetBaseDir(), clashPidFile)
}

func saveClashPid(pid int) error {
	return os.WriteFile(getClashPidFilePath(), []byte(fmt.Sprintf("%d", pid)), 0644)
}

func readClashPid() (int, error) {
	data, err := os.ReadFile(getClashPidFilePath())
	if err != nil {
		return 0, err
	}
	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	return pid, err
}

func clearClashPid() {
	os.Remove(getClashPidFilePath())
}

func (c *Core) IsInstalled() bool {
	_, err := os.Stat(config.CoreBinaryPath())
	return err == nil
}

func (c *Core) Install() error {
	if c.IsInstalled() { return nil }
	config.EnsureCoreDir()

	tmp := filepath.Join(config.GetBaseDir(), "mihomo.gz")
	if err := downloadFile(getCoreDownloadURL(), tmp); err != nil { return err }

	if err := ungzip(tmp, config.CoreBinaryPath()); err != nil { return err }
	os.Remove(tmp)
	os.Chmod(config.CoreBinaryPath(), 0755)
	return nil
}

func (c *Core) NeedsCapability() bool {
	cmd := exec.Command("getcap", config.CoreBinaryPath())
	output, _ := cmd.Output()
	return !strings.Contains(string(output), "cap_net_admin")
}

func (c *Core) SetCapability() error {
	cmd := exec.Command("sudo", "setcap", "cap_net_admin+ep", config.CoreBinaryPath())
	return cmd.Run()
}

func (c *Core) DownloadGeoData() error {
	mmdb := filepath.Join(config.GetBaseDir(), "Country.mmdb")
	geosite := filepath.Join(config.GetBaseDir(), "geosite.dat")

	if _, err := os.Stat(mmdb); os.IsNotExist(err) {
		if err := downloadFile(mmdbDownloadURL, mmdb); err != nil { return err }
	}
	if _, err := os.Stat(geosite); os.IsNotExist(err) {
		if err := downloadFile(geositeDownloadURL, geosite); err != nil { return err }
	}
	return nil
}

func (c *Core) Start() error {
	// Kill any existing process first
	c.Stop()

	cmd := exec.Command(config.CoreBinaryPath(), "-d", config.GetBaseDir())
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	// Save cmd for proper cleanup within same process
	runningCmd = cmd

	// Save PID to file for cross-process tracking
	saveClashPid(cmd.Process.Pid)

	return nil
}

func (c *Core) Stop() error {
	// Method 1: Kill tracked process in same process instance
	if runningCmd != nil && runningCmd.Process != nil {
		runningCmd.Process.Kill()
		runningCmd.Process.Wait() // Reap the zombie (only works for child)
		runningCmd = nil
		clearClashPid()
		return nil
	}

	// Method 2: Kill by PID file (for cross-process cleanup)
	pid, err := readClashPid()
	if err == nil && pid > 0 {
		process, _ := os.FindProcess(pid)
		process.Kill()
		clearClashPid()
	}

	// Method 3: Fallback - kill by process name (backup cleanup)
	exec.Command("pkill", "-f", "mihomo").Run()
	exec.Command("pkill", "-f", "clash -d "+config.GetBaseDir()).Run()
	time.Sleep(200 * time.Millisecond)

	return nil
}

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil { return err }
	defer resp.Body.Close()

	out, err := os.Create(dst)
	if err != nil { return err }
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func ungzip(src, dst string) error {
	f, err := os.Open(src)
	if err != nil { return err }
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil { return err }
	defer gr.Close()

	out, err := os.Create(dst)
	if err != nil { return err }
	defer out.Close()

	_, err = io.Copy(out, gr)
	return err
}

func DownloadSubscription(subURL string, proxyPort, apiPort int) ([]byte, SubscriptionInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(subURL)
	if err != nil { return nil, SubscriptionInfo{}, fmt.Errorf("fetch: %w", err) }
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return nil, SubscriptionInfo{}, fmt.Errorf("status: %d", resp.StatusCode) }

	data, _ := io.ReadAll(resp.Body)
	s := strings.TrimSpace(string(data))

	decoded, _ := base64.StdEncoding.DecodeString(s)
	lines := strings.Split(string(decoded), "\n")

	var nodes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "trojan://") ||
			strings.HasPrefix(line, "vless://") ||
			strings.HasPrefix(line, "hysteria2://") ||
			strings.HasPrefix(line, "hy2://") {
			nodes = append(nodes, line)
		}
	}

	info := parseSubscriptionInfo(subURL, resp.Header)

	configData := []byte(buildConfig(nodes, proxyPort, apiPort))
	if err := config.SaveConfig(configData); err != nil { return nil, SubscriptionInfo{}, err }
	if err := config.SaveSubscription(subURL); err != nil { return nil, SubscriptionInfo{}, err }
	return data, info, nil
}

func buildConfig(nodes []string, proxyPort, apiPort int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("mixed-port: %d\nallow-lan: true\nmode: rule\nlog-level: info\nexternal-controller: 127.0.0.1:%d\n", proxyPort, apiPort))

	names := []string{}
	realNodes := []string{}
	for _, n := range nodes {
		name := extractNodeName(n)
		if name != "" {
			names = append(names, name)
			realNodes = append(realNodes, n)
		}
	}

	if len(names) == 0 {
		for i := range nodes {
			names = append(names, fmt.Sprintf("Node%d", i+1))
		}
		realNodes = nodes
	}

	b.WriteString("\nproxies:\n")
	for _, n := range realNodes {
		b.WriteString(parseNodeConfig(n))
	}

	b.WriteString("\nproxy-groups:\n  - name: Auto\n    type: url-test\n    proxies:\n")
	for _, name := range names {
		b.WriteString(fmt.Sprintf("      - \"%s\"\n", name))
	}
	b.WriteString("    url: http://www.gstatic.com/generate_204\n    interval: 300\n  - name: Proxy\n    type: select\n    proxies:\n      - Auto\n")
	for _, name := range names {
		b.WriteString(fmt.Sprintf("      - \"%s\"\n", name))
	}
	b.WriteString("\nrules:\n  - MATCH,Proxy\n")
	return b.String()
}

func extractNodeName(link string) string {
	if strings.Contains(link, "#") {
		fragment := strings.SplitN(link, "#", 2)[1]
		decoded, err := url.QueryUnescape(fragment)
		if err == nil && decoded != "" {
			if strings.Contains(decoded, "流量") || strings.Contains(decoded, "到期") ||
				strings.Contains(decoded, "重置") || strings.Contains(decoded, "建议") {
				return ""
			}
			return decoded
		}
	}
	return "Node"
}

func parseNodeConfig(link string) string {
	name := extractNodeName(link)
	if name == "" {
		return ""
	}

	// Remove fragment from link for parsing
	if strings.Contains(link, "#") {
		link = strings.SplitN(link, "#", 2)[0]
	}

	if strings.HasPrefix(link, "trojan://") {
		link = strings.TrimPrefix(link, "trojan://")
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 { return "" }
		pass := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 { return "" }
		host, port := hp[0], hp[1]
		sni := host
		skip := false
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" { sni = q.Get("sni") }
			if q.Get("allowInsecure") == "1" { skip = true }
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: trojan\n    server: %s\n    port: %s\n    password: %s\n    sni: %s\n", name, host, port, pass, sni)
		if skip { r += "    skip-cert-verify: true\n" }
		return r
	}

	if strings.HasPrefix(link, "vless://") {
		link = strings.TrimPrefix(link, "vless://")
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 { return "" }
		uuid := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 { return "" }
		host, port := hp[0], hp[1]
		sni := host
		net := "tcp"
		skip := false
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" { sni = q.Get("sni") }
			if q.Get("type") != "" { net = q.Get("type") }
			if q.Get("allowInsecure") == "1" { skip = true }
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: vless\n    server: %s\n    port: %s\n    uuid: %s\n    network: %s\n    tls: true\n    servername: %s\n", name, host, port, uuid, net, sni)
		if skip { r += "    skip-cert-verify: true\n" }
		return r
	}

	if strings.HasPrefix(link, "hysteria2://") || strings.HasPrefix(link, "hy2://") {
		link = strings.TrimPrefix(link, "hysteria2://")
		link = strings.TrimPrefix(link, "hy2://")
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 { return "" }
		pass := p[0]
		hp := strings.SplitN(strings.TrimSuffix(strings.SplitN(p[1], "?", 2)[0], "/"), ":", 2)
		if len(hp) != 2 { return "" }
		host, port := hp[0], hp[1]
		sni := host
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" { sni = q.Get("sni") }
		}
		return fmt.Sprintf("  - name: \"%s\"\n    type: hysteria2\n    server: %s\n    port: %s\n    password: %s\n    sni: %s\n", name, host, port, pass, sni)
	}

	return ""
}