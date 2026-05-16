package clash

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
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
	"sync"
	"syscall"
	"time"

	"clashtui/internal/config"
	"clashtui/internal/geo"
	"clashtui/internal/validation"
)

const coreVersion = "v1.18.10"
const mmdbDownloadURL = "https://gh-proxy.com/https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb"
const geositeDownloadURL = "https://gh-proxy.com/https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat"

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

type Core struct {
	runningCmd *exec.Cmd
	mu         sync.Mutex
}

const clashPidFile = "clash.pid"

func NewCore() *Core {
	c := &Core{}
	c.cleanupStaleProcess()
	return c
}

func (c *Core) cleanupStaleProcess() {
	pid, err := readClashPid()
	if err != nil || pid == 0 {
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		clearClashPid()
		return
	}

	running := false
	if err := process.Signal(syscall.Signal(0)); err == nil {
		running = true
	}

	if running {
		process.Kill()
		process.Wait()
		clearClashPid()
	} else {
		clearClashPid()
	}
}

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
	if c.IsInstalled() {
		return nil
	}
	config.EnsureCoreDir()

	tmp := filepath.Join(config.GetBaseDir(), "mihomo.gz")
	if err := downloadFile(getCoreDownloadURL(), tmp); err != nil {
		return err
	}

	if err := ungzip(tmp, config.CoreBinaryPath()); err != nil {
		return err
	}
	os.Remove(tmp)
	os.Chmod(config.CoreBinaryPath(), 0755)
	return nil
}

func (c *Core) NeedsCapability() bool {
	if _, err := exec.LookPath("getcap"); err != nil {
		return false
	}

	cmd := exec.Command("getcap", config.RealCoreBinaryPath())
	output, _ := cmd.Output()
	return !strings.Contains(string(output), "cap_net_admin")
}

func (c *Core) HasGetcap() bool {
	_, err := exec.LookPath("getcap")
	return err == nil
}

func (c *Core) SetCapability() error {
	cmd := exec.Command("sudo", "setcap", "cap_net_admin+ep", config.CoreBinaryPath())
	return cmd.Run()
}

func (c *Core) DownloadGeoData() error {
	mmdb := filepath.Join(config.GetBaseDir(), "Country.mmdb")
	geosite := filepath.Join(config.GetBaseDir(), "geosite.dat")

	if _, err := os.Stat(mmdb); os.IsNotExist(err) {
		if err := geo.ExtractMMDB(mmdb); err != nil {
			return err
		}
	}
	if _, err := os.Stat(geosite); os.IsNotExist(err) {
		if err := geo.ExtractGeosite(geosite); err != nil {
			return err
		}
	}
	return nil
}

func (c *Core) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	configData, err := config.LoadConfigNoValidation()
	if err != nil {
		return fmt.Errorf("no config file, press 's' to add subscription")
	}

	if err := validation.ValidateConfig(configData); err != nil {
		return fmt.Errorf("config invalid: %w", err)
	}

	c.stopInternal()

	cmd := exec.Command(config.CoreBinaryPath(), "-d", config.GetBaseDir())
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return err
	}

	c.runningCmd = cmd
	saveClashPid(cmd.Process.Pid)

	return nil
}

func (c *Core) StartAndCheck() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	configData, err := config.LoadConfigNoValidation()
	if err != nil {
		return fmt.Errorf("no config file, press 's' to add subscription")
	}

	if err := validation.ValidateConfig(configData); err != nil {
		return fmt.Errorf("config invalid: %w", err)
	}

	c.stopInternal()

	stderrFile := filepath.Join(config.GetBaseDir(), "clash.err")
	f, err := os.Create(stderrFile)
	if err != nil {
		return err
	}

	cmd := exec.Command(config.CoreBinaryPath(), "-d", config.GetBaseDir())
	cmd.Stdout = nil
	cmd.Stderr = f

	if err := cmd.Start(); err != nil {
		f.Close()
		return err
	}

	c.runningCmd = cmd
	saveClashPid(cmd.Process.Pid)

	f.Close()

	if cmd.Process == nil {
		return fmt.Errorf("process failed to start")
	}

	if _, err := os.FindProcess(cmd.Process.Pid); err != nil {
		return fmt.Errorf("cannot find process: %v", err)
	}

	client := NewClient(9090)
	for i := 0; i < 10; i++ {
		time.Sleep(200 * time.Millisecond)
		if client.IsConnected() {
			return nil
		}
	}

	errData, _ := os.ReadFile(stderrFile)
	errMsg := string(errData)
	if strings.Contains(errMsg, "fatal") || strings.Contains(errMsg, "error") {
		lines := strings.Split(errMsg, "\n")
		for _, line := range lines {
			if strings.Contains(line, "fatal") || strings.Contains(line, "Parse config error") {
				return fmt.Errorf("core startup failed: %s", strings.TrimSpace(line))
			}
		}
		return fmt.Errorf("core startup failed, check %s", stderrFile)
	}
	return fmt.Errorf("core not responding at API port 9090")
}

func (c *Core) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopInternal()
}

// stopInternal is the unlocked version, called by Start and Stop
func (c *Core) stopInternal() error {
	// Method 1: Kill tracked process in same process instance
	if c.runningCmd != nil && c.runningCmd.Process != nil {
		c.runningCmd.Process.Kill()
		c.runningCmd.Process.Wait() // Reap the zombie
		c.runningCmd = nil
		clearClashPid()
		return nil
	}

	// Method 2: Kill by PID file (for cross-process cleanup)
	pid, err := readClashPid()
	if err == nil && pid > 0 {
		process, _ := os.FindProcess(pid)
		process.Kill()
		process.Wait() // Reap the zombie
		clearClashPid()
		return nil
	}

	// Method 3: Fallback - kill by process name with config path (more specific)
	exec.Command("pkill", "-f", "clash -d "+config.GetBaseDir()).Run()
	time.Sleep(200 * time.Millisecond)

	return nil
}

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func ungzip(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gr)
	return err
}

func DownloadSubscription(subURL string, proxyPort, apiPort int, tunMode bool) ([]byte, SubscriptionInfo, error) {
	if subURL == "" {
		return nil, SubscriptionInfo{}, fmt.Errorf("empty subscription URL")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", subURL, nil)
	if err != nil {
		return nil, SubscriptionInfo{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "ClashforWindows/0.20.39")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, SubscriptionInfo{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, SubscriptionInfo{}, fmt.Errorf("status: %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	s := strings.TrimSpace(string(data))

	info := parseSubscriptionInfo(subURL, resp.Header)

	var configData []byte

	if strings.HasPrefix(s, "proxies:") || strings.HasPrefix(s, "mixed-port:") {
		configData = replaceDNSInConfig(data, proxyPort, apiPort)
	} else {
		nodes := parseSubscriptionContent(s)
		if len(nodes) == 0 {
			return nil, info, fmt.Errorf("no valid nodes found in subscription")
		}
		configData = []byte(buildConfig(nodes, proxyPort, apiPort, tunMode))
	}

	if err := validation.ValidateConfig(configData); err != nil {
		return nil, info, fmt.Errorf("config validation failed: %w", err)
	}

	if err := config.SaveConfig(configData); err != nil {
		return nil, SubscriptionInfo{}, err
	}
	return data, info, nil
}

func parseSubscriptionContent(content string) []string {
	s := strings.TrimSpace(content)
	var nodes []string

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err == nil && len(decoded) > 0 {
		lines := strings.Split(string(decoded), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if isValidNodeLink(line) {
				nodes = append(nodes, line)
			}
		}
		if len(nodes) > 0 {
			return nodes
		}
	}

	decodedRaw, err := base64.RawStdEncoding.DecodeString(s)
	if err == nil && len(decodedRaw) > 0 {
		lines := strings.Split(string(decodedRaw), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if isValidNodeLink(line) {
				nodes = append(nodes, line)
			}
		}
		if len(nodes) > 0 {
			return nodes
		}
	}

	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if isValidNodeLink(line) {
			nodes = append(nodes, line)
		}
	}

	return nodes
}

// replaceDNSInConfig replaces DNS section in clash config
func replaceDNSInConfig(configData []byte, proxyPort, apiPort int) []byte {
	content := string(configData)

	// Simple DNS config - disable to use system DNS
	safeDNS := `dns:
  enable: false`

	// Find and replace DNS section
	// DNS section starts with "dns:" and ends at next top-level key (line not starting with space)
	lines := strings.Split(content, "\n")
	var result []string
	skipDNS := false
	dnsReplaced := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect DNS section start
		if trimmed == "dns:" || strings.HasPrefix(trimmed, "dns:") {
			skipDNS = true
			// Insert our safe DNS config
			result = append(result, safeDNS)
			dnsReplaced = true
			continue
		}

		// Skip lines in DNS section (they start with space or are continuation)
		if skipDNS {
			// Check if we've reached a new top-level section
			// Top-level keys don't start with space and aren't empty
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
				skipDNS = false
				result = append(result, line)
			}
			// Otherwise skip this line (it's part of old DNS config)
			continue
		}

		result = append(result, line)

		// Also update mixed-port and external-controller if present
		if strings.HasPrefix(trimmed, "mixed-port:") && i+1 < len(lines) {
			// Keep original mixed-port if it exists
		}
	}

	// If no DNS section found, add it after mixed-port
	if !dnsReplaced {
		// Find where to insert DNS (after mixed-port/allow-lan/basic settings)
		insertIdx := 0
		for i := range result {
			trimmed := strings.TrimSpace(result[i])
			if strings.HasPrefix(trimmed, "mixed-port:") ||
				strings.HasPrefix(trimmed, "allow-lan:") ||
				strings.HasPrefix(trimmed, "mode:") ||
				strings.HasPrefix(trimmed, "log-level:") {
				insertIdx = i + 1
			}
			// Stop at first non-basic setting
			if strings.HasPrefix(trimmed, "proxies:") ||
				strings.HasPrefix(trimmed, "proxy-groups:") ||
				strings.HasPrefix(trimmed, "rules:") {
				break
			}
		}
		if insertIdx > 0 {
			// Insert DNS config
			newResult := result[:insertIdx]
			newResult = append(newResult, safeDNS)
			newResult = append(newResult, result[insertIdx:]...)
			result = newResult
		}
	}

	return []byte(strings.Join(result, "\n"))
}

func isValidNodeLink(link string) bool {
	return strings.HasPrefix(link, "trojan://") ||
		strings.HasPrefix(link, "vless://") ||
		strings.HasPrefix(link, "vmess://") ||
		strings.HasPrefix(link, "ss://") ||
		strings.HasPrefix(link, "ssr://") ||
		strings.HasPrefix(link, "hysteria2://") ||
		strings.HasPrefix(link, "hy2://") ||
		strings.HasPrefix(link, "hysteria://") ||
		strings.HasPrefix(link, "anytls://") ||
		strings.HasPrefix(link, "socks5://") ||
		strings.HasPrefix(link, "socks://") ||
		strings.HasPrefix(link, "http://") ||
		strings.HasPrefix(link, "https://") ||
		strings.HasPrefix(link, "wireguard://") ||
		strings.HasPrefix(link, "tuic://") ||
		strings.HasPrefix(link, "ssh://")
}

// ContainsNodeLinks 检查内容是否包含节点链接
func ContainsNodeLinks(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if isValidNodeLink(line) {
			return true
		}
	}
	// 也检查 base64 编码的内容
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content))
	if err == nil {
		for _, line := range strings.Split(string(decoded), "\n") {
			if isValidNodeLink(strings.TrimSpace(line)) {
				return true
			}
		}
	}
	decodedRaw, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(content))
	if err == nil {
		for _, line := range strings.Split(string(decodedRaw), "\n") {
			if isValidNodeLink(strings.TrimSpace(line)) {
				return true
			}
		}
	}
	return false
}

// ParseNodeLinks 从内容中解析节点链接
func ParseNodeLinks(content string) []string {
	return parseSubscriptionContent(content)
}

// BuildConfigFromNodes 从节点链接构建配置文件
func BuildConfigFromNodes(nodes []string, proxyPort, apiPort int, tunMode bool) string {
	return buildConfig(nodes, proxyPort, apiPort, tunMode)
}

func ProcessConfigForTUN(configData []byte, enableTUN bool) []byte {
	content := string(configData)

	tunConfig := `tun:
  enable: true
  stack: system
  auto-route: true
  auto-detect-interface: true
  dns-hijack:
    - any:53`

	lines := strings.Split(content, "\n")
	var result []string
	skipTUN := false
	tunFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "tun:" || strings.HasPrefix(trimmed, "tun:") {
			skipTUN = true
			tunFound = true
			if enableTUN {
				result = append(result, tunConfig)
			}
			continue
		}

		if skipTUN {
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
				skipTUN = false
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)
	}

	if enableTUN && !tunFound {
		for i, line := range result {
			if strings.TrimSpace(line) == "enable: false" && i > 0 {
				if strings.Contains(result[i-1], "dns:") {
					newResult := result[:i+1]
					newResult = append(newResult, tunConfig)
					newResult = append(newResult, result[i+1:]...)
					result = newResult
					break
				}
			}
		}
	}

	return []byte(strings.Join(result, "\n"))
}

func buildConfig(nodes []string, proxyPort, apiPort int, tunMode bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("mixed-port: %d\nallow-lan: true\nmode: rule\nlog-level: info\nexternal-controller: 127.0.0.1:%d\n", proxyPort, apiPort))

	b.WriteString("\ndns:\n  enable: false\n")

	if tunMode {
		b.WriteString("\ntun:\n  enable: true\n  stack: system\n  auto-route: true\n  auto-detect-interface: true\n  dns-hijack:\n    - any:53\n")
	}

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
		if len(p) != 2 {
			return ""
		}
		pass := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		sni := host
		skip := false
		grpc := false
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" {
				sni = q.Get("sni")
			}
			if q.Get("peer") != "" {
				sni = q.Get("peer")
			}
			if q.Get("allowInsecure") == "1" || q.Get("insecure") == "1" {
				skip = true
			}
			if q.Get("type") == "grpc" {
				grpc = true
			}
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: trojan\n    server: %s\n    port: %s\n    password: %s\n    sni: %s\n", name, host, port, pass, sni)
		if skip {
			r += "    skip-cert-verify: true\n"
		}
		if grpc {
			r += "    network: grpc\n"
			if serviceName := ""; serviceName != "" {
				r += fmt.Sprintf("    grpc-opts:\n      grpc-service-name: %s\n", serviceName)
			}
		}
		return r
	}

	if strings.HasPrefix(link, "vless://") {
		link = strings.TrimPrefix(link, "vless://")
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		uuid := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		sni := host
		net := "tcp"
		skip := false
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" {
				sni = q.Get("sni")
			}
			if q.Get("type") != "" {
				net = q.Get("type")
			}
			if q.Get("allowInsecure") == "1" {
				skip = true
			}
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: vless\n    server: %s\n    port: %s\n    uuid: %s\n    network: %s\n    tls: true\n    servername: %s\n", name, host, port, uuid, net, sni)
		if skip {
			r += "    skip-cert-verify: true\n"
		}
		return r
	}

	if strings.HasPrefix(link, "anytls://") {
		link = strings.TrimPrefix(link, "anytls://")
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		uuid := p[0]
		hp := strings.SplitN(strings.TrimSuffix(strings.SplitN(p[1], "?", 2)[0], "/"), ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		sni := host
		net := "tcp"
		fp := ""
		skip := false
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" {
				sni = q.Get("sni")
			}
			if q.Get("type") != "" {
				net = q.Get("type")
			}
			if q.Get("fp") != "" {
				fp = q.Get("fp")
			}
			if q.Get("insecure") == "1" {
				skip = true
			}
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: vless\n    server: %s\n    port: %s\n    uuid: %s\n    network: %s\n    tls: true\n    udp: true\n    flow: xtls-rprx-vision\n    servername: %s\n", name, host, port, uuid, net, sni)
		if fp != "" {
			r += fmt.Sprintf("    client-fingerprint: %s\n", fp)
		}
		if skip {
			r += "    skip-cert-verify: true\n"
		}
		return r
	}

	if strings.HasPrefix(link, "hysteria2://") || strings.HasPrefix(link, "hy2://") {
		link = strings.TrimPrefix(link, "hysteria2://")
		link = strings.TrimPrefix(link, "hy2://")
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		pass := p[0]
		hp := strings.SplitN(strings.TrimSuffix(strings.SplitN(p[1], "?", 2)[0], "/"), ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		sni := host
		skip := false
		var ports string
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			if q.Get("sni") != "" {
				sni = q.Get("sni")
			}
			if q.Get("insecure") == "1" {
				skip = true
			}
			if q.Get("mport") != "" {
				ports = q.Get("mport")
			}
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: hysteria2\n    server: %s\n    port: %s\n    password: %s\n    sni: %s\n", name, host, port, pass, sni)
		if ports != "" {
			r += fmt.Sprintf("    ports: %s\n", ports)
		}
		if skip {
			r += "    skip-cert-verify: true\n"
		}
		return r
	}

	if strings.HasPrefix(link, "vmess://") {
		link = strings.TrimPrefix(link, "vmess://")
		decoded, err := base64.StdEncoding.DecodeString(link)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(link)
			if err != nil {
				return ""
			}
		}
		var vm map[string]interface{}
		if err := json.Unmarshal(decoded, &vm); err != nil {
			return ""
		}
		server, _ := vm["add"].(string)
		port, _ := vm["port"].(string)
		uuid, _ := vm["id"].(string)
		net, _ := vm["net"].(string)
		if net == "" {
			net = "tcp"
		}
		tls, _ := vm["tls"].(string)
		hostHeader, _ := vm["host"].(string)
		r := fmt.Sprintf("  - name: \"%s\"\n    type: vmess\n    server: %s\n    port: %s\n    uuid: %s\n    network: %s\n", name, server, port, uuid, net)
		if tls == "tls" {
			r += "    tls: true\n"
		}
		if hostHeader != "" && net == "ws" {
			r += fmt.Sprintf("    ws-headers:\n      Host: %s\n", hostHeader)
		}
		return r
	}

	if strings.HasPrefix(link, "ss://") {
		link = strings.TrimPrefix(link, "ss://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}

		if strings.Contains(link, "@") {
			p := strings.SplitN(link, "@", 2)
			if len(p) != 2 {
				return ""
			}

			methodPass := p[0]
			hostPort := p[1]

			var method, pass string
			d, err := base64.StdEncoding.DecodeString(methodPass)
			if err != nil {
				d, err = base64.RawStdEncoding.DecodeString(methodPass)
				if err != nil {
					mp := strings.SplitN(methodPass, ":", 2)
					if len(mp) != 2 {
						return ""
					}
					method, pass = mp[0], mp[1]
				} else {
					mp := strings.SplitN(string(d), ":", 2)
					if len(mp) != 2 {
						return ""
					}
					method, pass = mp[0], mp[1]
				}
			} else {
				mp := strings.SplitN(string(d), ":", 2)
				if len(mp) != 2 {
					return ""
				}
				method, pass = mp[0], mp[1]
			}

			hp := strings.SplitN(hostPort, ":", 2)
			if len(hp) != 2 {
				return ""
			}
			return fmt.Sprintf("  - name: \"%s\"\n    type: ss\n    server: %s\n    port: %s\n    cipher: %s\n    password: %s\n", name, hp[0], hp[1], method, pass)
		}
		return ""
	}

	if strings.HasPrefix(link, "hysteria://") {
		link = strings.TrimPrefix(link, "hysteria://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		pass := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		return fmt.Sprintf("  - name: \"%s\"\n    type: hysteria\n    server: %s\n    port: %s\n    auth_str: %s\n", name, host, port, pass)
	}

	// SSR (ShadowsocksR)
	if strings.HasPrefix(link, "ssr://") {
		link = strings.TrimPrefix(link, "ssr://")
		decoded, err := base64.StdEncoding.DecodeString(link)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(link)
			if err != nil {
				decoded = []byte(link)
			}
		}
		// SSR format: server:port:protocol:method:obfs:password_base64/?obfsparam=...&protoparam=...&remarks=...
		parts := strings.Split(string(decoded), "/?")
		if len(parts) < 1 {
			return ""
		}
		mainParts := strings.Split(parts[0], ":")
		if len(mainParts) < 6 {
			return ""
		}
		host, port, protocol, method, obfs := mainParts[0], mainParts[1], mainParts[2], mainParts[3], mainParts[4]
		passDecoded, err := base64.StdEncoding.DecodeString(mainParts[5])
		if err != nil {
			passDecoded, err = base64.RawStdEncoding.DecodeString(mainParts[5])
			if err != nil {
				passDecoded = []byte(mainParts[5])
			}
		}
		password := string(passDecoded)
		return fmt.Sprintf("  - name: \"%s\"\n    type: ssr\n    server: %s\n    port: %s\n    cipher: %s\n    password: %s\n    protocol: %s\n    obfs: %s\n", name, host, port, method, password, protocol, obfs)
	}

	// SOCKS5
	if strings.HasPrefix(link, "socks5://") || strings.HasPrefix(link, "socks://") {
		link = strings.TrimPrefix(link, "socks5://")
		link = strings.TrimPrefix(link, "socks://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}
		// Format: user:pass@host:port or host:port
		var user, pass, host, port string
		if strings.Contains(link, "@") {
			p := strings.SplitN(link, "@", 2)
			up := strings.SplitN(p[0], ":", 2)
			if len(up) == 2 {
				user, pass = up[0], up[1]
			}
			hp := strings.SplitN(p[1], ":", 2)
			if len(hp) == 2 {
				host, port = hp[0], hp[1]
			}
		} else {
			hp := strings.SplitN(link, ":", 2)
			if len(hp) == 2 {
				host, port = hp[0], hp[1]
			}
		}
		if host == "" || port == "" {
			return ""
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: socks5\n    server: %s\n    port: %s\n", name, host, port)
		if user != "" {
			r += fmt.Sprintf("    username: %s\n    password: %s\n", user, pass)
		}
		return r
	}

	// HTTP Proxy
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		isTLS := strings.HasPrefix(link, "https://")
		link = strings.TrimPrefix(link, "http://")
		link = strings.TrimPrefix(link, "https://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}
		// Format: user:pass@host:port or host:port
		var user, pass, host, port string
		if strings.Contains(link, "@") {
			p := strings.SplitN(link, "@", 2)
			up := strings.SplitN(p[0], ":", 2)
			if len(up) == 2 {
				user, pass = up[0], up[1]
			}
			hp := strings.SplitN(p[1], ":", 2)
			if len(hp) == 2 {
				host, port = hp[0], hp[1]
			}
		} else {
			hp := strings.SplitN(link, ":", 2)
			if len(hp) == 2 {
				host, port = hp[0], hp[1]
			}
		}
		if host == "" || port == "" {
			return ""
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: http\n    server: %s\n    port: %s\n", name, host, port)
		if user != "" {
			r += fmt.Sprintf("    username: %s\n    password: %s\n", user, pass)
		}
		if isTLS {
			r += "    tls: true\n"
		}
		return r
	}

	// WireGuard - basic support
	if strings.HasPrefix(link, "wireguard://") {
		link = strings.TrimPrefix(link, "wireguard://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}
		// Format: privateKey@host:port?publicKey=...&reserved=...
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		privateKey := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		var publicKey, reserved string
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			publicKey = q.Get("publicKey")
			reserved = q.Get("reserved")
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: wireguard\n    server: %s\n    port: %s\n    private-key: %s\n", name, host, port, privateKey)
		if publicKey != "" {
			r += fmt.Sprintf("    peer-public-key: %s\n", publicKey)
		}
		if reserved != "" {
			r += fmt.Sprintf("    reserved: [%s]\n", reserved)
		}
		return r
	}

	// TUIC - basic support
	if strings.HasPrefix(link, "tuic://") {
		link = strings.TrimPrefix(link, "tuic://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}
		// Format: uuid:password@host:port?congestion_control=...&alpn=...
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		up := strings.SplitN(p[0], ":", 2)
		if len(up) != 2 {
			return ""
		}
		uuid, password := up[0], up[1]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		var cc, alpn string
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			cc = q.Get("congestion_control")
			alpn = q.Get("alpn")
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: tuic\n    server: %s\n    port: %s\n    uuid: %s\n    password: %s\n", name, host, port, uuid, password)
		if cc != "" {
			r += fmt.Sprintf("    congestion-controller: %s\n", cc)
		}
		if alpn != "" {
			r += fmt.Sprintf("    alpn:\n      - %s\n", alpn)
		}
		return r
	}

	// SSH - basic support
	if strings.HasPrefix(link, "ssh://") {
		link = strings.TrimPrefix(link, "ssh://")
		if strings.Contains(link, "#") {
			link = strings.SplitN(link, "#", 2)[0]
		}
		// Format: user@host:port?privateKey=...
		p := strings.SplitN(link, "@", 2)
		if len(p) != 2 {
			return ""
		}
		user := p[0]
		hp := strings.SplitN(strings.SplitN(p[1], "?", 2)[0], ":", 2)
		if len(hp) != 2 {
			return ""
		}
		host, port := hp[0], hp[1]
		var privateKey string
		if strings.Contains(p[1], "?") {
			q, _ := url.ParseQuery(strings.SplitN(p[1], "?", 2)[1])
			privateKey = q.Get("privateKey")
		}
		r := fmt.Sprintf("  - name: \"%s\"\n    type: ssh\n    server: %s\n    port: %s\n    username: %s\n", name, host, port, user)
		if privateKey != "" {
			r += fmt.Sprintf("    private-key: %s\n", privateKey)
		}
		return r
	}

	return ""
}
