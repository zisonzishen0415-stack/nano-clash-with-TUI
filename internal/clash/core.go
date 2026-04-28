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
	"strings"
	"time"

	"clashtui/internal/config"
)

const coreDownloadURL = "https://gh-proxy.com/https://github.com/MetaCubeX/mihomo/releases/download/v1.18.10/mihomo-linux-amd64-v1.18.10.gz"
const mmdbDownloadURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
const geositeDownloadURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"

type Core struct{}

func NewCore() *Core { return &Core{} }

func (c *Core) IsInstalled() bool {
	_, err := os.Stat(config.CoreBinaryPath())
	return err == nil
}

func (c *Core) Install() error {
	if c.IsInstalled() { return nil }
	config.EnsureCoreDir()
	
	tmp := filepath.Join(config.GetBaseDir(), "mihomo.gz")
	if err := downloadFile(coreDownloadURL, tmp); err != nil { return err }
	
	if err := ungzip(tmp, config.CoreBinaryPath()); err != nil { return err }
	os.Remove(tmp)
	os.Chmod(config.CoreBinaryPath(), 0755)
	return nil
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
	cmd := exec.Command(config.CoreBinaryPath(), "-d", config.GetBaseDir())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func (c *Core) Stop() error {
	exec.Command("pkill", "-f", "clash -d "+config.GetBaseDir()).Run()
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

func DownloadSubscription(subURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(subURL)
	if err != nil { return nil, fmt.Errorf("fetch: %w", err) }
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return nil, fmt.Errorf("status: %d", resp.StatusCode) }

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

	configData := []byte(buildConfig(nodes))
	if err := config.SaveConfig(configData); err != nil { return nil, err }
	if err := config.SaveSubscription(subURL); err != nil { return nil, err }
	return configData, nil
}

func buildConfig(nodes []string) string {
	var b strings.Builder
	b.WriteString("mixed-port: 7890\nallow-lan: true\nmode: rule\nlog-level: info\nexternal-controller: 127.0.0.1:9090\n\nproxies:\n")
	for i, n := range nodes { b.WriteString(parseNode(n, i)) }
	b.WriteString("\nproxy-groups:\n  - name: Auto\n    type: url-test\n    proxies:\n")
	for i := range nodes { b.WriteString(fmt.Sprintf("      - Node%d\n", i+1)) }
	b.WriteString("    url: http://www.gstatic.com/generate_204\n    interval: 300\n  - name: Proxy\n    type: select\n    proxies:\n      - Auto\n")
	for i := range nodes { b.WriteString(fmt.Sprintf("      - Node%d\n", i+1)) }
	b.WriteString("\nrules:\n  - MATCH,Proxy\n")
	return b.String()
}

func parseNode(link string, i int) string {
	i++
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
		r := fmt.Sprintf("  - name: Node%d\n    type: trojan\n    server: %s\n    port: %s\n    password: %s\n    sni: %s\n", i, host, port, pass, sni)
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
		r := fmt.Sprintf("  - name: Node%d\n    type: vless\n    server: %s\n    port: %s\n    uuid: %s\n    network: %s\n    tls: true\n    servername: %s\n", i, host, port, uuid, net, sni)
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
		return fmt.Sprintf("  - name: Node%d\n    type: hysteria2\n    server: %s\n    port: %s\n    password: %s\n    sni: %s\n", i, host, port, pass, sni)
	}
	return ""
}
