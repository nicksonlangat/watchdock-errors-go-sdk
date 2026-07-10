package watchdock

import (
	"context"
	"errors"
	"testing"
)

func TestCaptureErrorWithContextSanitizesAndMergesScope(t *testing.T) {
	t.Cleanup(resetGlobal)

	var captured *Event

	err := Init(Config{
		APIKey:      "wdk_test_key",
		Environment: "test",
		Release:     "1.2.3",
		ServerName:  "test-server",
		SendPII:     false,
		BeforeSend: func(event Event) (*Event, error) {
			copy := event
			captured = &copy
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx := WithScope(context.Background(), Scope{
		Request: &RequestData{
			Method: "POST",
			URL:    "/checkout",
			Headers: map[string]string{
				"Authorization":   "Bearer secret",
				"X-Forwarded-For": "1.2.3.4",
			},
			Body: map[string]interface{}{
				"card": "4242424242424242",
			},
		},
		User: &UserData{
			ID:       "123",
			Email:    "user@example.com",
			Username: "nick",
		},
	})

	CaptureErrorWithContext(ctx, errors.New("boom"), &CaptureContext{
		Title: "Handled error",
	})

	if captured == nil {
		t.Fatal("expected event to be captured in BeforeSend")
	}

	if captured.ProjectKey != "wdk_test_key" {
		t.Fatalf("ProjectKey = %q, want %q", captured.ProjectKey, "wdk_test_key")
	}

	if captured.Title != "Handled error" {
		t.Fatalf("Title = %q, want %q", captured.Title, "Handled error")
	}

	if captured.Environment != "test" {
		t.Fatalf("Environment = %q, want %q", captured.Environment, "test")
	}

	if captured.Release != "1.2.3" {
		t.Fatalf("Release = %q, want %q", captured.Release, "1.2.3")
	}

	if captured.Request == nil {
		t.Fatal("expected request to be present")
	}

	if got := captured.Request.Headers["Authorization"]; got != "[REDACTED]" {
		t.Fatalf("Authorization header = %q, want redacted", got)
	}

	if got := captured.Request.Headers["X-Forwarded-For"]; got != "[REDACTED]" {
		t.Fatalf("X-Forwarded-For header = %q, want redacted", got)
	}

	if got := captured.Request.Body; got != "[REDACTED]" {
		t.Fatalf("Body = %#v, want redacted", got)
	}

	if captured.User == nil {
		t.Fatal("expected user to be present")
	}

	if captured.User.Email != "" {
		t.Fatalf("User.Email = %q, want empty when SendPII=false", captured.User.Email)
	}

	if captured.User.ID != "123" || captured.User.Username != "nick" {
		t.Fatalf("User = %#v, want ID and Username preserved", captured.User)
	}
}

func TestCaptureMessageWithPIIEnabledPreservesPII(t *testing.T) {
	t.Cleanup(resetGlobal)

	var captured *Event

	err := Init(Config{
		APIKey:      "wdk_test_key",
		Environment: "test",
		SendPII:     true,
		BeforeSend: func(event Event) (*Event, error) {
			copy := event
			captured = &copy
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	ctx := WithScope(context.Background(), Scope{
		Request: &RequestData{
			Headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4",
			},
			Body: "plain-body",
		},
		User: &UserData{
			ID:    "123",
			Email: "user@example.com",
		},
	})

	CaptureMessageWithContext(ctx, "hello", nil)

	if captured == nil {
		t.Fatal("expected event to be captured in BeforeSend")
	}

	if got := captured.Request.Headers["X-Forwarded-For"]; got != "1.2.3.4" {
		t.Fatalf("X-Forwarded-For header = %q, want original value", got)
	}

	if got := captured.Request.Body; got != "plain-body" {
		t.Fatalf("Body = %#v, want original string", got)
	}

	if captured.User == nil || captured.User.Email != "user@example.com" {
		t.Fatalf("User.Email = %#v, want preserved email", captured.User)
	}
}

func TestCaptureErrorDefaultsToErrorLevel(t *testing.T) {
	t.Cleanup(resetGlobal)

	var captured *Event

	err := Init(Config{
		APIKey: "wdk_test_key",
		BeforeSend: func(event Event) (*Event, error) {
			copy := event
			captured = &copy
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	CaptureError(errors.New("boom"), nil)

	if captured == nil {
		t.Fatal("expected event to be captured in BeforeSend")
	}

	if captured.Level != "error" {
		t.Fatalf("Level = %q, want %q", captured.Level, "error")
	}
}

func TestCaptureMessageDefaultsToInfoLevelAndAcceptsOverride(t *testing.T) {
	t.Cleanup(resetGlobal)

	var captured *Event

	err := Init(Config{
		APIKey: "wdk_test_key",
		BeforeSend: func(event Event) (*Event, error) {
			copy := event
			captured = &copy
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	CaptureMessage("disk almost full", nil)

	if captured == nil {
		t.Fatal("expected event to be captured in BeforeSend")
	}

	if captured.Level != "info" {
		t.Fatalf("Level = %q, want %q", captured.Level, "info")
	}

	CaptureMessage("disk almost full", &CaptureContext{Level: "warning"})

	if captured.Level != "warning" {
		t.Fatalf("Level = %q, want %q", captured.Level, "warning")
	}
}

func resetGlobal() {
	globalMu.Lock()
	defer globalMu.Unlock()

	if global != nil {
		close(global.queue)
		global = nil
	}
}
