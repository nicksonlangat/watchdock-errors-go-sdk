package watchdock

import "testing"

func TestBaseURL(t *testing.T) {
	tests := []struct {
		endpoint string
		want     string
	}{
		{"https://api.watchdock.cc/api/v1/error-events/", "https://api.watchdock.cc"},
		{"https://api.watchdock.cc/api/v1/error-events", "https://api.watchdock.cc"},
		{"http://127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"http://127.0.0.1:8080/", "http://127.0.0.1:8080"},
	}

	for _, tt := range tests {
		if got := baseURL(tt.endpoint); got != tt.want {
			t.Errorf("baseURL(%q) = %q, want %q", tt.endpoint, got, tt.want)
		}
	}
}

func TestExtractTraceID(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"x-request-id", map[string]string{"X-Request-Id": "abc123"}, "abc123"},
		{
			"traceparent fallback",
			map[string]string{"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},
			"4bf92f3577b34da6a3ce929d0e0e4736",
		},
		{"x-request-id wins over traceparent", map[string]string{
			"X-Request-Id": "abc123",
			"traceparent":  "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		}, "abc123"},
		{"no matching header", map[string]string{"Content-Type": "application/json"}, ""},
		{"nil headers", nil, ""},
	}

	for _, tt := range tests {
		if got := extractTraceID(tt.headers); got != tt.want {
			t.Errorf("%s: extractTraceID(%v) = %q, want %q", tt.name, tt.headers, got, tt.want)
		}
	}
}
