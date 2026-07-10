package watchdock

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	sdkName    = "watchdock-errors-go-sdk"
	sdkVersion = "0.1.0"
)

func buildException(err error) Exception {
	return Exception{
		Type:       errorType(err),
		Message:    err.Error(),
		Stacktrace: callers(),
	}
}

func errorType(err error) string {
	return fmt.Sprintf("%T", err)
}

func callers() []StackFrame {
	pcs := make([]uintptr, 32)
	count := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:count])

	stack := make([]StackFrame, 0, count)
	for {
		frame, more := frames.Next()
		stack = append(stack, StackFrame{
			Filename:   frame.File,
			Function:   frame.Function,
			LineNumber: frame.Line,
		})
		if !more {
			break
		}
	}

	return stack
}

func buildServerData(serverName string, override *ServerData) *ServerData {
	server := &ServerData{
		Hostname:       firstNonEmpty(overrideValue(override, func(v *ServerData) string { return v.Hostname }), serverName, hostname()),
		Runtime:        firstNonEmpty(overrideValue(override, func(v *ServerData) string { return v.Runtime }), "go"),
		RuntimeVersion: firstNonEmpty(overrideValue(override, func(v *ServerData) string { return v.RuntimeVersion }), runtime.Version()),
		Platform:       firstNonEmpty(overrideValue(override, func(v *ServerData) string { return v.Platform }), fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)),
	}
	return server
}

func mergeContext(scope Scope, capture *CaptureContext) *CaptureContext {
	merged := &CaptureContext{}
	if capture != nil {
		*merged = *capture
	}
	if merged.Request == nil {
		merged.Request = scope.Request
	}
	if merged.User == nil {
		merged.User = scope.User
	}
	if merged.Server == nil {
		merged.Server = scope.Server
	}
	return merged
}

func sanitizeEvent(event *Event, sendPII bool) {
	if event.Request != nil {
		event.Request.Headers = sanitizeHeaders(event.Request.Headers, sendPII)
		if !sendPII && event.Request.Body != nil {
			event.Request.Body = "[REDACTED]"
		}
	}

	if !sendPII && event.User != nil {
		event.User = &UserData{
			ID:       event.User.ID,
			Username: event.User.Username,
		}
	}
}

func sanitizeHeaders(headers map[string]string, sendPII bool) map[string]string {
	if headers == nil {
		return nil
	}

	sanitized := make(map[string]string, len(headers))
	for key, value := range headers {
		switch strings.ToLower(key) {
		case "authorization", "cookie", "set-cookie":
			sanitized[key] = "[REDACTED]"
		case "x-forwarded-for":
			if sendPII {
				sanitized[key] = value
			} else {
				sanitized[key] = "[REDACTED]"
			}
		default:
			sanitized[key] = value
		}
	}

	return sanitized
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func overrideValue[T any](ptr *T, getter func(*T) string) string {
	if ptr == nil {
		return ""
	}
	return getter(ptr)
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// baseURL strips the ingest API path from an endpoint, leaving just the
// platform base URL (e.g. "https://api.watchdock.cc/api/v1/error-events/"
// becomes "https://api.watchdock.cc"), so other platform routes can be built
// from it.
func baseURL(endpoint string) string {
	trimmed := strings.TrimSuffix(endpoint, "/")
	if idx := strings.Index(trimmed, "/api/v1/"); idx != -1 {
		return trimmed[:idx]
	}
	return trimmed
}
