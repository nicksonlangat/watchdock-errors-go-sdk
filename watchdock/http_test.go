package watchdock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareCapturesUnhandledPanic(t *testing.T) {
	t.Cleanup(resetGlobal)

	received := make(chan Event, 1)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}

		received <- event
		w.WriteHeader(http.StatusAccepted)
	}))
	defer api.Close()

	err := Init(Config{
		APIKey:     "wdk_test_key",
		Endpoint:   api.URL,
		HTTPClient: api.Client(),
		SendPII:    false,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("middleware panic")
	}))

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("expected panic to be re-raised")
			}
		}()

		req := httptest.NewRequest(http.MethodGet, "/panic?source=test", nil)
		req.Header.Set("Authorization", "Bearer secret")
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}()

	Flush()

	select {
	case event := <-received:
		if event.Title != "Unhandled HTTP panic" {
			t.Fatalf("Title = %q, want %q", event.Title, "Unhandled HTTP panic")
		}

		if event.Exception.Message != "middleware panic" {
			t.Fatalf("Exception.Message = %q, want %q", event.Exception.Message, "middleware panic")
		}

		if event.Request == nil {
			t.Fatal("expected request to be present")
		}

		if got := event.Request.Headers["Authorization"]; got != "[REDACTED]" {
			t.Fatalf("Authorization header = %q, want redacted", got)
		}

		if got := event.Request.Headers["X-Forwarded-For"]; got != "[REDACTED]" {
			t.Fatalf("X-Forwarded-For header = %q, want redacted", got)
		}

		if event.Request.Method != http.MethodGet {
			t.Fatalf("Method = %q, want %q", event.Request.Method, http.MethodGet)
		}
	default:
		t.Fatal("expected middleware to send an event")
	}
}
