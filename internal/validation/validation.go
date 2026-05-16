package validation

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func ValidateYAML(configData []byte) error {
	if len(configData) == 0 {
		return ValidationError{Message: "config is empty"}
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(configData, &raw); err != nil {
		return ValidationError{
			Field:   "yaml syntax",
			Message: fmt.Sprintf("invalid YAML syntax: %v", err),
		}
	}

	return nil
}

func ValidateStructure(configData []byte) error {
	if len(configData) == 0 {
		return ValidationError{Message: "config is empty or missing"}
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(configData, &raw); err != nil {
		return ValidationError{
			Field:   "yaml",
			Message: fmt.Sprintf("parse error: %v", err),
		}
	}

	requiredSections := []string{"proxies", "proxy-groups", "rules"}

	for _, section := range requiredSections {
		if _, exists := raw[section]; !exists {
			return ValidationError{
				Field:   section,
				Message: fmt.Sprintf("missing required section: %s", section),
			}
		}
	}

	return nil
}

func ValidateConfig(configData []byte) error {
	if err := ValidateYAML(configData); err != nil {
		return fmt.Errorf("syntax validation failed: %w", err)
	}

	if err := ValidateStructure(configData); err != nil {
		return fmt.Errorf("structure validation failed: %w", err)
	}

	return nil
}

func GetValidationErrorMessage(err error) string {
	var suggestions []string

	if ve, ok := err.(ValidationError); ok {
		switch ve.Field {
		case "proxies":
			suggestions = []string{
				"Check subscription URL is valid",
				"Try importing nodes manually",
			}
		case "proxy-groups":
			suggestions = []string{
				"Subscription may be incomplete",
				"Delete and re-add subscription",
			}
		case "rules":
			suggestions = []string{
				"Subscription missing rules section",
				"Use default config with imported nodes",
			}
		case "yaml syntax":
			suggestions = []string{
				"Check for indentation errors",
				"Subscription may have corrupt data",
			}
		default:
			if strings.Contains(ve.Message, "empty") {
				suggestions = []string{
					"No config file found",
					"Press 's' to add subscription or 'c' to import from clipboard",
				}
			} else {
				suggestions = []string{
					"Delete subscription and re-add",
					"Or import nodes manually",
				}
			}
		}

		msg := ve.Error()
		if len(suggestions) > 0 {
			msg += "\n  Suggestion: " + suggestions[0]
		}
		return msg
	}

	return err.Error() + "\n  Suggestion: Check config or re-download subscription"
}
