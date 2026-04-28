package clash

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"clashtui/internal/config"
)

const coreDownloadURL = "https://gh-proxy.com/https://github.com/MetaCubeX/mihomo/releases/download/v1.18.10/mihomo-linux-amd64-v1.18.10.gz"
const mmdbDownloadURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb"
const geositeDownloadURL = "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat"

type Core struct {
	cmd     *exec.Cmd
	running bool
}

func NewCore() *Core {
	return &Core{}
}

func (c *Core) IsInstalled() bool {
	_, err := os.Stat(config.CoreBinaryPath())
	return err == nil
}

func (c *Core) IsRunning() bool {
	return c.running
}

func (c *Core) Install() error {
	if err := config.EnsureCoreDir(); err != nil {
		return err
	}

	tmpFile := filepath.Join(config.GetBaseDir(), "clash.gz")
	if err := downloadFile(coreDownloadURL, tmpFile); err != nil {
		return fmt.Errorf("download core: %w", err)
	}

	if err := decompressGzip(tmpFile, config.CoreBinaryPath()); err != nil {
		return fmt.Errorf("decompress core: %w", err)
	}

	os.Remove(tmpFile)
	os.Chmod(config.CoreBinaryPath(), 0755)

	return nil
}

func (c *Core) DownloadGeoData() error {
	baseDir := config.GetBaseDir()

	mmdbPath := filepath.Join(baseDir, "Country.mmdb")
	geositePath := filepath.Join(baseDir, "geosite.dat")

	if _, err := os.Stat(mmdbPath); os.IsNotExist(err) {
		if err := downloadFile(mmdbDownloadURL, mmdbPath); err != nil {
			return fmt.Errorf("download mmdb: %w", err)
		}
	}

	if _, err := os.Stat(geositePath); os.IsNotExist(err) {
		if err := downloadFile(geositeDownloadURL, geositePath); err != nil {
			return fmt.Errorf("download geosite: %w", err)
		}
	}

	return nil
}

func (c *Core) Start() error {
	baseDir := config.GetBaseDir()

	c.cmd = exec.Command(config.CoreBinaryPath(), "-d", baseDir)
	c.cmd.Stdout = nil
	c.cmd.Stderr = nil

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start core: %w", err)
	}

	c.running = true
	time.Sleep(2 * time.Second)

	return nil
}

func (c *Core) Stop() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}

	if err := c.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill core: %w", err)
	}

	c.running = false
	return nil
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func decompressGzip(src, dst string) error {
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

func DownloadSubscription(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("subscription error: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := config.SaveConfig(data); err != nil {
		return nil, err
	}

	if err := config.SaveSubscription(url); err != nil {
		return nil, err
	}

	return data, nil
}