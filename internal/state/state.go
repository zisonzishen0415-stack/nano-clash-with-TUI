package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"clashtui/internal/config"
)

type NetworkMode string

const (
	ModeOff NetworkMode = "off"
	ModeTUN NetworkMode = "tun"
)

type NetworkState struct {
	Mode      NetworkMode `json:"mode"`
	Port      int         `json:"port,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

func GetStateFilePath() string {
	return filepath.Join(config.GetBaseDir(), "network-state.json")
}

func SaveState(state NetworkState) error {
	state.Timestamp = time.Now().Unix()

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	stateFile := GetStateFilePath()
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func LoadState() NetworkState {
	stateFile := GetStateFilePath()

	data, err := os.ReadFile(stateFile)
	if os.IsNotExist(err) {
		return NetworkState{Mode: ModeOff}
	}
	if err != nil {
		return NetworkState{Mode: ModeOff}
	}

	var state NetworkState
	if err := json.Unmarshal(data, &state); err != nil {
		return NetworkState{Mode: ModeOff}
	}

	if state.Mode == "" {
		return NetworkState{Mode: ModeOff}
	}

	return state
}

func ClearState() error {
	stateFile := GetStateFilePath()
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
