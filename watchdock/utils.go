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
	sdkVersion = "0.3.0"
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

// contextLines is the number of source lines captured on each side of a
// frame's failing line. Matches the window the backend and the other SDKs use.
const contextLines = 2

func callers() []StackFrame {
	pcs := make([]uintptr, 32)
	count := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:count])

	stack := make([]StackFrame, 0, count)
	// Read each source file at most once per capture: a stack usually has
	// several frames from the same file, and a nil entry remembers a miss so we
	// don't keep stat-ing a file that isn't on disk.
	cache := map[string][]string{}
	for {
		frame, more := frames.Next()
		contextLine, pre, post := sourceContext(frame.File, frame.Line, cache)
		stack = append(stack, StackFrame{
			Filename:    frame.File,
			Function:    frame.Function,
			LineNumber:  frame.Line,
			ContextLine: contextLine,
			PreContext:  pre,
			PostContext: post,
		})
		if !more {
			break
		}
	}

	return stack
}

// sourceContext returns the failing line plus the lines just before and after
// it, when the source file is readable. Go records the compile-time path in
// each frame, so this resolves in development and anywhere the source ships
// with the binary; on a stripped production host it returns empty values and
// the frame keeps just its location. Out-of-range lines come back as empty
// strings so the pre/post slices are always contextLines long.
func sourceContext(file string, line int, cache map[string][]string) (string, []string, []string) {
	if file == "" || line <= 0 {
		return "", nil, nil
	}

	lines, seen := cache[file]
	if !seen {
		data, err := os.ReadFile(file)
		if err != nil {
			cache[file] = nil // remember the miss
			return "", nil, nil
		}
		lines = strings.Split(string(data), "\n")
		cache[file] = lines
	}
	if lines == nil {
		return "", nil, nil
	}

	idx := line - 1 // frame line numbers are 1-based
	if idx < 0 || idx >= len(lines) {
		return "", nil, nil
	}

	at := func(n int) string { // n is a 1-based line number
		if n >= 1 && n-1 < len(lines) {
			return strings.TrimRight(lines[n-1], "\r")
		}
		return ""
	}

	pre := make([]string, 0, contextLines)
	for i := line - contextLines; i < line; i++ {
		pre = append(pre, at(i))
	}
	post := make([]string, 0, contextLines)
	for i := line + 1; i <= line+contextLines; i++ {
		post = append(post, at(i))
	}

	return strings.TrimSpace(at(line)), pre, post
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

// extractTraceID pulls a correlation ID off incoming request headers so this
// event can be linked back to the nginx access log line for the same
// request. Prefers X-Request-Id (nginx's built-in $request_id, zero extra
// modules required) and falls back to the trace-id segment of a W3C
// traceparent header if present.
func extractTraceID(headers map[string]string) string {
	if headers == nil {
		return ""
	}

	for key, value := range headers {
		if strings.EqualFold(key, "x-request-id") && value != "" {
			return value
		}
	}

	for key, value := range headers {
		if strings.EqualFold(key, "traceparent") && value != "" {
			parts := strings.Split(value, "-")
			if len(parts) >= 2 && parts[1] != "" {
				return parts[1]
			}
		}
	}

	return ""
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
