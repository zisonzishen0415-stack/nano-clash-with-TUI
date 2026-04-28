package clipboard

import (
	"errors"
	"os/exec"
	"strings"
)

func Read() (string, error) {
	if content, err := tryXclip(); err == nil {
		return content, nil
	}
	if content, err := tryXsel(); err == nil {
		return content, nil
	}
	return "", errors.New("no clipboard tool available (install xclip or xsel)")
}

func tryXclip() (string, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func tryXsel() (string, error) {
	cmd := exec.Command("xsel", "--clipboard", "--output")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func Available() bool {
	_, err := Read()
	return err == nil
}