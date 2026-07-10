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
