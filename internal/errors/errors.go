package errors

import (
	"fmt"
	"strings"
)

type ErrorType int

const (
	ErrorTypeCritical ErrorType = iota
	ErrorTypeOperational
	ErrorTypeWarning
)

type ClassifiedError struct {
	Type        ErrorType
	Message     string
	Context     string
	Recovery    string
	AutoCleanup bool
}

func (e ClassifiedError) Error() string {
	return e.Message
}

func NewCriticalError(message, context, recovery string) ClassifiedError {
	return ClassifiedError{
		Type:        ErrorTypeCritical,
		Message:     message,
		Context:     context,
		Recovery:    recovery,
		AutoCleanup: true,
	}
}

func NewOperationalError(message, context, recovery string) ClassifiedError {
	return ClassifiedError{
		Type:        ErrorTypeOperational,
		Message:     message,
		Context:     context,
		Recovery:    recovery,
		AutoCleanup: false,
	}
}

func NewWarningError(message string) ClassifiedError {
	return ClassifiedError{
		Type:        ErrorTypeWarning,
		Message:     message,
		Context:     "",
		Recovery:    "",
		AutoCleanup: false,
	}
}

func (e ClassifiedError) FullMessage() string {
	var parts []string

	parts = append(parts, e.Message)

	if e.Context != "" {
		parts = append(parts, "  Context: "+e.Context)
	}

	if e.Recovery != "" {
		parts = append(parts, "  Recovery: "+e.Recovery)
	}

	return strings.Join(parts, "\n")
}

func (e ClassifiedError) Icon() string {
	switch e.Type {
	case ErrorTypeCritical:
		return "⚠"
	case ErrorTypeOperational:
		return "⚠"
	case ErrorTypeWarning:
		return "⚡"
	default:
		return "⚠"
	}
}

func ConfigValidationError(err error) ClassifiedError {
	return NewOperationalError(
		"Config validation failed",
		err.Error(),
		"Delete subscription and re-add, or import nodes manually",
	)
}

func CoreStartError(err error, stderrPath string) ClassifiedError {
	return NewCriticalError(
		"Core failed to start",
		err.Error(),
		fmt.Sprintf("Run 'clashtui --restore-network' if network broken. Check logs at %s", stderrPath),
	)
}

func TUNCapabilityError(binaryPath string) ClassifiedError {
	return NewOperationalError(
		"TUN mode needs capability",
		"cap_net_admin capability missing",
		fmt.Sprintf("Run: sudo setcap cap_net_admin+ep %s", binaryPath),
	)
}

func SubscriptionDownloadError(statusCode int, url string) ClassifiedError {
	return NewOperationalError(
		"Subscription download failed",
		fmt.Sprintf("HTTP status %d from %s", statusCode, url),
		"Check subscription URL or try refresh again",
	)
}

func NetworkBrokenError() ClassifiedError {
	return NewCriticalError(
		"Network connectivity broken",
		"Proxy settings inconsistent",
		"Run 'clashtui --restore-network' to fix",
	)
}

func PreserveErrorContext(err error, stderrFile string) ClassifiedError {
	baseMsg := err.Error()

	if strings.Contains(baseMsg, "validation") {
		return ConfigValidationError(err)
	}

	if strings.Contains(baseMsg, "capability") || strings.Contains(baseMsg, "TUN") {
		return TUNCapabilityError(stderrFile)
	}

	if strings.Contains(baseMsg, "status") || strings.Contains(baseMsg, "download") {
		return SubscriptionDownloadError(0, "")
	}

	if strings.Contains(baseMsg, "core") || strings.Contains(baseMsg, "API port") {
		return CoreStartError(err, stderrFile)
	}

	return NewOperationalError(baseMsg, "", "Stop core (x) and try restart, or run --restore-network")
}
