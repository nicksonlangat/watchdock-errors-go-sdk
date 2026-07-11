package watchdock

type StackFrame struct {
	Filename    string `json:"filename"`
	Function    string `json:"function,omitempty"`
	LineNumber  int    `json:"lineno,omitempty"`
	ContextLine string `json:"context_line,omitempty"`
}

type Exception struct {
	Type       string       `json:"type"`
	Message    string       `json:"message"`
	Stacktrace []StackFrame `json:"stacktrace,omitempty"`
}

type RequestData struct {
	Method      string                 `json:"method,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string]interface{} `json:"query_params,omitempty"`
	Body        interface{}            `json:"body,omitempty"`
}

type UserData struct {
	ID       string `json:"id,omitempty"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
}

type ServerData struct {
	Hostname       string `json:"hostname,omitempty"`
	Runtime        string `json:"runtime,omitempty"`
	RuntimeVersion string `json:"runtime_version,omitempty"`
	Platform       string `json:"platform,omitempty"`
}

type SDKData struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Event struct {
	ProjectKey  string       `json:"project_key,omitempty"`
	Title       string       `json:"title,omitempty"`
	Timestamp   string       `json:"timestamp"`
	Environment string       `json:"environment,omitempty"`
	Level       string       `json:"level,omitempty"`
	Release     string       `json:"release,omitempty"`
	TraceID     string       `json:"trace_id,omitempty"`
	Exception   Exception    `json:"exception"`
	Request     *RequestData `json:"request,omitempty"`
	User        *UserData    `json:"user,omitempty"`
	Server      *ServerData  `json:"server,omitempty"`
	SDK         SDKData      `json:"sdk"`
}

type CaptureContext struct {
	Title       string
	Environment string
	Level       string
	Release     string
	// TraceID correlates this event with the originating nginx request
	// (e.g. X-Request-Id). Takes priority over any value auto-extracted
	// from Request.Headers if both are present.
	TraceID string
	Request *RequestData
	User    *UserData
	Server  *ServerData
}
