package health

import (
	"strings"

	"clashtui/internal/clash"
	"clashtui/internal/config"
	"clashtui/internal/settings"
	"clashtui/internal/state"
)

type HealthCheckResult struct {
	Issue       string
	ActionTaken string
}

func CheckCoreRunning(apiPort int) bool {
	client := clash.NewClient(apiPort)
	return client.IsConnected()
}

func CheckTUNState() bool {
	if !config.Exists() {
		return false
	}

	data, err := config.LoadConfigNoValidation()
	if err != nil {
		return false
	}

	content := string(data)
	return strings.Contains(content, "tun:") && strings.Contains(content, "enable: true")
}

func RunHealthChecks(s settings.Settings) []HealthCheckResult {
	results := []HealthCheckResult{}
	coreRunning := CheckCoreRunning(s.APIPort)

	if s.TUNMode && !coreRunning {
		data, _ := config.LoadConfigNoValidation()
		newData := clash.ProcessConfigForTUN(data, false)
		config.SaveConfig(newData)
		state.SaveState(state.NetworkState{Mode: state.ModeOff})
		results = append(results, HealthCheckResult{
			Issue:       "stale TUN mode",
			ActionTaken: "disabled TUN in config",
		})
	}

	return results
}
