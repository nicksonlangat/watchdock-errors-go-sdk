# watchdock-errors-go-sdk

Watchdock error tracking SDK for Go backends.

## Features

- Async delivery to Watchdock ingest API
- Manual `CaptureError` and `CaptureMessage`
- Configurable event `Level` (defaults to `"error"` for errors, `"info"` for messages)
- `net/http` middleware for unhandled panics
- Request/user context support

## Install

```bash
go get github.com/nicksonlangat/watchdock-errors-go-sdk
```

## Quick Start

```go
package main

import (
    "errors"

    watchdock "github.com/nicksonlangat/watchdock-errors-go-sdk/watchdock"
)

func main() {
    watchdock.Init(watchdock.Config{
        APIKey:      "wdk_your_tracking_key",
        Environment: "production",
        Release:     "1.0.0",
        ServerName:  "api-1",
    })

    watchdock.CaptureError(errors.New("payment provider rejected request"), nil)
    watchdock.CaptureMessage("background sync completed with warnings", nil)

    // Override the default level via CaptureContext
    watchdock.CaptureMessage("queue depth high", &watchdock.CaptureContext{
        Level: "warning",
    })
}
```

### Event levels

Every event carries a `Level`. `CaptureError`/`CaptureErrorWithContext` default to `"error"`; `CaptureMessage`/`CaptureMessageWithContext` default to `"info"`. Override either by setting `CaptureContext.Level`.

## SDK initialization

`Init()` schedules a one-time, fire-and-forget ping to the platform (with the SDK version and environment) to register that the SDK started up. This runs in a background goroutine, never blocks `Init`, and any failure is silently ignored.

## net/http Middleware

```go
mux := http.NewServeMux()
handler := watchdock.Middleware(mux)
http.ListenAndServe(":8080", handler)
```

## Demo

See [`demo/main.go`](./demo/main.go).

Run it with:

```bash
cp demo/.env.example demo/.env
go run ./demo
```
