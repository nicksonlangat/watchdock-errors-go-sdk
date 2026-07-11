package watchdock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

type client struct {
	config Config
	queue  chan Event
	wg     sync.WaitGroup
}

var (
	globalMu sync.RWMutex
	global   *client
)

func Init(config Config) error {
	if !strings.HasPrefix(config.APIKey, "wdk_") {
		return errors.New("watchdock API key must start with 'wdk_'")
	}

	if config.Endpoint == "" {
		config.Endpoint = defaultEndpoint
	}

	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 3 * time.Second}
	}

	c := &client{
		config: config,
		queue:  make(chan Event, 100),
	}

	go c.loop()

	globalMu.Lock()
	defer globalMu.Unlock()

	if global != nil {
		close(global.queue)
	}

	global = c
	sendInitPing(config)
	return nil
}

func CaptureError(err error, capture *CaptureContext) {
	if err == nil {
		return
	}

	sendEvent(Event{
		Title:       captureTitle(err, capture),
		Environment: captureValue(func() string { return capture.Environment }),
		Level:       captureLevel(capture, "error"),
		Release:     captureValue(func() string { return capture.Release }),
		TraceID:     captureValue(func() string { return capture.TraceID }),
		Exception:   buildException(err),
	}, capture)
}

func CaptureMessage(message string, capture *CaptureContext) {
	sendEvent(Event{
		Title:       firstNonEmpty(captureValue(func() string { return capture.Title }), message),
		Environment: captureValue(func() string { return capture.Environment }),
		Level:       captureLevel(capture, "info"),
		Release:     captureValue(func() string { return capture.Release }),
		TraceID:     captureValue(func() string { return capture.TraceID }),
		Exception: Exception{
			Type:    "Message",
			Message: message,
		},
	}, capture)
}

func CaptureErrorWithContext(ctx context.Context, err error, capture *CaptureContext) {
	sendEventWithContext(ctx, Event{
		Title:       captureTitle(err, capture),
		Environment: captureValue(func() string { return capture.Environment }),
		Level:       captureLevel(capture, "error"),
		Release:     captureValue(func() string { return capture.Release }),
		TraceID:     captureValue(func() string { return capture.TraceID }),
		Exception:   buildException(err),
	}, capture)
}

func CaptureMessageWithContext(ctx context.Context, message string, capture *CaptureContext) {
	sendEventWithContext(ctx, Event{
		Title:       firstNonEmpty(captureValue(func() string { return capture.Title }), message),
		Environment: captureValue(func() string { return capture.Environment }),
		Level:       captureLevel(capture, "info"),
		Release:     captureValue(func() string { return capture.Release }),
		TraceID:     captureValue(func() string { return capture.TraceID }),
		Exception: Exception{
			Type:    "Message",
			Message: message,
		},
	}, capture)
}

func Flush() {
	globalMu.RLock()
	c := global
	globalMu.RUnlock()

	if c == nil {
		return
	}

	c.wg.Wait()
}

func sendEvent(event Event, capture *CaptureContext) {
	sendEventWithContext(context.Background(), event, capture)
}

func sendEventWithContext(ctx context.Context, event Event, capture *CaptureContext) {
	globalMu.RLock()
	c := global
	globalMu.RUnlock()
	if c == nil {
		return
	}

	scope, _ := ScopeFromContext(ctx)
	merged := mergeContext(scope, capture)

	event.ProjectKey = c.config.APIKey
	event.Timestamp = nowISO()
	event.Environment = firstNonEmpty(event.Environment, c.config.Environment, "production")
	event.Release = firstNonEmpty(event.Release, c.config.Release)
	event.Request = merged.Request
	event.User = merged.User
	event.Server = buildServerData(c.config.ServerName, merged.Server)
	event.SDK = SDKData{Name: sdkName, Version: sdkVersion}
	if event.TraceID == "" && merged.Request != nil {
		event.TraceID = extractTraceID(merged.Request.Headers)
	}

	sanitizeEvent(&event, c.config.SendPII)

	if c.config.BeforeSend != nil {
		next, err := c.config.BeforeSend(event)
		if err != nil {
			c.reportError(err)
			return
		}
		if next == nil {
			return
		}
		event = *next
	}

	c.wg.Add(1)
	select {
	case c.queue <- event:
	default:
		c.wg.Done()
		c.reportError(errors.New("watchdock event queue is full"))
	}
}

func (c *client) loop() {
	for event := range c.queue {
		c.post(event)
		c.wg.Done()
	}
}

func (c *client) post(event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		c.reportError(err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, c.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		c.reportError(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		c.reportError(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.reportError(errors.New("watchdock ingestion failed with status " + resp.Status))
	}
}

func (c *client) reportError(err error) {
	if c.config.OnError != nil {
		c.config.OnError(err)
	}
}

func captureValue(getter func() string) string {
	defer func() {
		_ = recover()
	}()
	return getter()
}

func captureTitle(err error, capture *CaptureContext) string {
	if err == nil {
		return ""
	}
	if capture != nil && capture.Title != "" {
		return capture.Title
	}
	return err.Error()
}

func captureLevel(capture *CaptureContext, fallback string) string {
	return firstNonEmpty(captureValue(func() string { return capture.Level }), fallback)
}

// sendInitPing fires a fire-and-forget POST to register SDK initialisation
// with the platform. It waits before pinging to allow the server to be ready
// at startup, and never blocks Init or raises on failure.
func sendInitPing(config Config) {
	go func() {
		defer func() { _ = recover() }()

		time.Sleep(5 * time.Second)

		payload, err := json.Marshal(map[string]string{
			"sdk_version": sdkVersion,
			"environment": config.Environment,
		})
		if err != nil {
			return
		}

		req, err := http.NewRequest(http.MethodPost, baseURL(config.Endpoint)+"/api/v1/errors/sdk-init/", bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+config.APIKey)

		httpClient := &http.Client{Timeout: 3 * time.Second}
		resp, err := httpClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}()
}
