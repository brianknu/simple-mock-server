package router

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"simple-mock-server/internal/mock"
	"time"
)

// RequestLogger is called after each request is handled.
// If nil is passed to RegisterMocks, requests are logged to the standard logger.
type RequestLogger interface {
	Log(entry RequestEntry)
}

type RequestEntry struct {
	Method       string
	Path         string
	Status       int
	RequestBody  string
	ResponseTime int
}

func RegisterMocks(mux *DynamicMux, mocks []mock.Mock, logger RequestLogger) {
	for _, m := range mocks {
		m := m // capture loop variable
		for _, path := range m.Paths {
			mux.Handle(path, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != m.Verb {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				for hk, hv := range m.Headers {
					w.Header().Set(hk, hv)
				}
				if m.ResponseTime > 0 {
					time.Sleep(time.Duration(m.ResponseTime) * time.Millisecond)
				}
				w.WriteHeader(m.Status)
				json.NewEncoder(w).Encode(m.Body)

				var bodyStr string
				if m.PrintRequestBody {
					if b, err := io.ReadAll(r.Body); err == nil {
						bodyStr = string(b)
					}
				}

				entry := RequestEntry{
					Method:       m.Verb,
					Path:         r.RequestURI,
					Status:       m.Status,
					RequestBody:  bodyStr,
					ResponseTime: m.ResponseTime,
				}

				if logger != nil {
					logger.Log(entry)
				} else {
					if bodyStr != "" {
						log.Printf("%s %s\n%s", entry.Method, entry.Path, bodyStr)
					} else {
						log.Printf("%s %s", entry.Method, entry.Path)
					}
				}
			})
		}
	}
}
