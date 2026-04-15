package watchdock

import "net/http"

const defaultEndpoint = "https://api.watchdock.cc/api/v1/error-events/"

type Config struct {
	APIKey      string
	Endpoint    string
	Environment string
	Release     string
	ServerName  string
	SendPII     bool
	HTTPClient  *http.Client
	BeforeSend  func(Event) (*Event, error)
	OnError     func(error)
}
