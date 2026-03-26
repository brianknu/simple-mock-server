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

// MockSelector is called when the user must choose among mocks.
// Return values:
//
//	>= 0  index of the selected mock
//	-1    cancel (respond with 404)
//	-2    define a new mock (triggers Fallback handler)
type MockSelector func(path string, verb string, mocks []mock.Mock) int

// RouteConfig holds the options for RegisterMocks.
type RouteConfig struct {
	Logger       RequestLogger
	Selector     MockSelector
	CaptureCheck func() bool      // returns true when capture mode is on
	Fallback     http.HandlerFunc // called when selector returns -2 (define new)
}

func RegisterMocks(mux *DynamicMux, mocks []mock.Mock, cfg RouteConfig) {
	// Group mocks by path
	grouped := make(map[string][]mock.Mock)

	for _, m := range mocks {
		if m.Disabled {
			continue
		}
		for _, path := range m.Paths {
			grouped[path] = append(grouped[path], m)
		}
	}

	for path, pathMocks := range grouped {
		pathMocks := pathMocks // capture
		path := path
		mux.Handle(path, func(w http.ResponseWriter, r *http.Request) {
			// Filter by verb
			var matches []mock.Mock
			for _, m := range pathMocks {
				if m.Verb == r.Method {
					matches = append(matches, m)
				}
			}

			captureOn := cfg.CaptureCheck != nil && cfg.CaptureCheck()

			if len(matches) == 0 {
				// No verb match — in capture mode, let user define a new mock
				if captureOn && cfg.Fallback != nil {
					cfg.Fallback(w, r)
					return
				}
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			// Single match and capture is off → serve directly
			if len(matches) == 1 && !captureOn {
				serveMock(w, r, matches[0], cfg.Logger)
				return
			}

			// Multiple matches, or capture mode is on → ask user to pick
			if cfg.Selector != nil {
				idx := cfg.Selector(path, r.Method, matches)
				switch {
				case idx == -2 && cfg.Fallback != nil:
					cfg.Fallback(w, r)
					return
				case idx < 0 || idx >= len(matches):
					http.NotFound(w, r)
					return
				default:
					serveMock(w, r, matches[idx], cfg.Logger)
					return
				}
			}

			// No selector (headless) → first match
			serveMock(w, r, matches[0], cfg.Logger)
		})
	}
}

func serveMock(w http.ResponseWriter, r *http.Request, m mock.Mock, logger RequestLogger) {
	for hk, hv := range m.Headers {
		w.Header().Set(hk, hv)
	}
	if m.ResponseTime > 0 {
		time.Sleep(time.Duration(m.ResponseTime) * time.Millisecond)
	}
	w.WriteHeader(m.Status)
	json.NewEncoder(w).Encode(m.Body)

	var bodyStr string
	if b, err := io.ReadAll(r.Body); err == nil {
		bodyStr = string(b)
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
}
