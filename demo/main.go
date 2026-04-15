package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	watchdock "github.com/nicksonlangat/watchdock-errors-go-sdk/watchdock"
)

func main() {
	_ = godotenv.Load("demo/.env")

	apiKey := os.Getenv("WATCHDOCK_API_KEY")
	if apiKey == "" {
		log.Fatal("Missing WATCHDOCK_API_KEY")
	}

	err := watchdock.Init(watchdock.Config{
		APIKey:      apiKey,
		Environment: "development",
		Release:     "demo-go-api@0.1.0",
		ServerName:  "demo-go-api",
		SendPII:     false,
		OnError: func(err error) {
			log.Println("watchdock send error:", err)
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("watchdock go demo api"))
	})

	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		watchdock.CaptureMessageWithContext(r.Context(), "background sync completed with warnings", &watchdock.CaptureContext{
			Title: "Demo custom message",
		})
		_, _ = w.Write([]byte("captured message"))
	})

	mux.HandleFunc("/handled-error", func(w http.ResponseWriter, r *http.Request) {
		watchdock.CaptureErrorWithContext(r.Context(), errors.New("handled demo error from go api"), &watchdock.CaptureContext{
			Title: "Handled demo error",
		})
		_, _ = w.Write([]byte("captured handled error"))
	})

	mux.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("unhandled demo panic from go api")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Demo API running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, watchdock.Middleware(mux)))
}
