package router

import (
	"net/http"
	"sync"
)

// DynamicMux is an HTTP request multiplexer that supports dynamic route
// registration and removal at runtime. It is safe for concurrent use.
type DynamicMux struct {
	mu       sync.RWMutex
	handlers map[string]http.HandlerFunc
}

func NewDynamicMux() *DynamicMux {
	return &DynamicMux{
		handlers: make(map[string]http.HandlerFunc),
	}
}

func (m *DynamicMux) Handle(pattern string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[pattern] = handler
}

func (m *DynamicMux) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = make(map[string]http.HandlerFunc)
}

func (m *DynamicMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	handler, ok := m.handlers[r.URL.Path]
	m.mu.RUnlock()

	if ok {
		handler(w, r)
	} else {
		http.NotFound(w, r)
	}
}
