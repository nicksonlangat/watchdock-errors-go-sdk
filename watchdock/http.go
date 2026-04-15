package watchdock

import (
	"fmt"
	"net/http"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := Scope{
			Request: requestFromHTTP(r),
		}
		ctx := WithScope(r.Context(), scope)

		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("%v", recovered)
				CaptureErrorWithContext(ctx, err, &CaptureContext{
					Title: "Unhandled HTTP panic",
				})
				panic(recovered)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestFromHTTP(r *http.Request) *RequestData {
	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	query := make(map[string]interface{}, len(r.URL.Query()))
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			query[key] = values[0]
		} else {
			query[key] = values
		}
	}

	return &RequestData{
		Method:      r.Method,
		URL:         r.URL.String(),
		Headers:     headers,
		QueryParams: query,
	}
}
