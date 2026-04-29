package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"clashtui/internal/config"
)

const SettingsFile = "settings.json"

type Subscription struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Traffic    string    `json:"traffic"`
	Expiry     string    `json:"expiry"`
	LastUpdate time.Time `json:"last_update"`
}

type Settings struct {
	Subscriptions  []Subscription `json:"subscriptions"`
	ActiveSubIdx   int            `json:"active_sub_idx"`
	AutoStart      bool           `json:"auto_start"`
	AutoTestDelay  bool           `json:"auto_test_delay"`
	AutoSelectBest bool           `json:"auto_select_best"`
	UseDefaultNode bool           `json:"use_default_node"`
	DefaultNode    string         `json:"default_node"`
	ProxyPort      int            `json:"proxy_port"`
	APIPort        int            `json:"api_port"`
}

var DefaultSettings = Settings{
	Subscriptions:  []Subscription{},
	ActiveSubIdx:   0,
	AutoStart:      false,
	AutoTestDelay:  true,
	AutoSelectBest: true,
	UseDefaultNode: false,
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

	if s.ProxyPort == 0 {
		s.ProxyPort = DefaultSettings.ProxyPort
	}
	if s.APIPort == 0 {
		s.APIPort = DefaultSettings.APIPort
	}
	if s.Subscriptions == nil {
		s.Subscriptions = []Subscription{}
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

func GetActiveSubscription(s Settings) *Subscription {
	if len(s.Subscriptions) == 0 {
		return nil
	}
	if s.ActiveSubIdx < 0 || s.ActiveSubIdx >= len(s.Subscriptions) {
		return nil
	}
	return &s.Subscriptions[s.ActiveSubIdx]
}

func AddSubscription(s *Settings, name, url string) {
	sub := Subscription{
		Name: name,
		URL:  url,
	}
	s.Subscriptions = append(s.Subscriptions, sub)
	Save(*s)
}

func RemoveSubscription(s *Settings, idx int) {
	if idx < 0 || idx >= len(s.Subscriptions) {
		return
	}
	s.Subscriptions = append(s.Subscriptions[:idx], s.Subscriptions[idx+1:]...)
	if s.ActiveSubIdx >= len(s.Subscriptions) {
		s.ActiveSubIdx = len(s.Subscriptions) - 1
	}
	if s.ActiveSubIdx < 0 {
		s.ActiveSubIdx = 0
	}
	Save(*s)
}

func SwitchSubscription(s *Settings, idx int) {
	if idx < 0 || idx >= len(s.Subscriptions) {
		return
	}
	s.ActiveSubIdx = idx
	Save(*s)
}

func MigrateFromOldFormat() {
	oldPath := config.GetOldSubscriptionPath()
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return
	}

	url := strings.TrimSpace(string(data))
	if url == "" {
		return
	}

	s := Load()
	if len(s.Subscriptions) > 0 {
		os.Remove(oldPath)
		return
	}

	AddSubscription(&s, "Migrated", url)
	s.ActiveSubIdx = 0
	Save(s)

	os.Remove(oldPath)
}