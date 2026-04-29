package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const SettingsFile = "settings.json"

type Settings struct {
	AutoStart      bool `json:"auto_start"`       // 开机自启动
	AutoTestDelay  bool `json:"auto_test_delay"`  // 自动测速
	AutoSelectBest bool `json:"auto_select_best"` // 自动选择最快节点
	DefaultNode    string `json:"default_node"`    // 默认节点（空=自动选择）
	ProxyPort      int  `json:"proxy_port"`       // 代理端口
	APIPort        int  `json:"api_port"`         // API 端口
}

var DefaultSettings = Settings{
	AutoStart:      false,
	AutoTestDelay:  true,
	AutoSelectBest: true,
	DefaultNode:    "",
	ProxyPort:      7890,
	APIPort:        9090,
}

var settingsPath string

func init() {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "clashtui")
	settingsPath = filepath.Join(baseDir, SettingsFile)
}

func Load() Settings {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return DefaultSettings
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings
	}

	// Apply defaults for missing fields
	if s.ProxyPort == 0 {
		s.ProxyPort = DefaultSettings.ProxyPort
	}
	if s.APIPort == 0 {
		s.APIPort = DefaultSettings.APIPort
	}

	return s
}

func Save(s Settings) error {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "clashtui")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

func GetSettingsPath() string {
	return settingsPath
}