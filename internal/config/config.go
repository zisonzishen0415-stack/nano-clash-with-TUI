package config

import (
	"os"
	"path/filepath"
)

const ConfigDir = ".config/clashTUI"
const ConfigFile = "config.yaml"
const SubscriptionFile = "subscriptions.txt"
const CoreDir = "core"

var configPath string
var subscriptionPath string
var corePath string

func init() {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ConfigDir)
	configPath = filepath.Join(baseDir, ConfigFile)
	subscriptionPath = filepath.Join(baseDir, SubscriptionFile)
	corePath = filepath.Join(baseDir, CoreDir)
}

func EnsureDir() error {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ConfigDir)
	return os.MkdirAll(baseDir, 0755)
}

func SaveConfig(content []byte) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return os.WriteFile(configPath, content, 0644)
}

func LoadConfig() ([]byte, error) {
	return os.ReadFile(configPath)
}

func ConfigExists() bool {
	_, err := os.Stat(configPath)
	return err == nil
}

func SaveSubscription(url string) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return os.WriteFile(subscriptionPath, []byte(url), 0644)
}

func LoadSubscription() (string, error) {
	data, err := os.ReadFile(subscriptionPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func CoreBinaryPath() string {
	return filepath.Join(corePath, "clash")
}

func EnsureCoreDir() error {
	return os.MkdirAll(corePath, 0755)
}

func GetBaseDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir)
}