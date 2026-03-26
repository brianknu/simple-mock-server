package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/router"
)

// PendingRequest represents an unmatched HTTP request waiting for the user
// to define a mock response via the TUI (capture mode).
type PendingRequest struct {
	Method     string
	Path       string
	Headers    http.Header
	Body       string
	ResponseCh chan PendingResponse
}

// PendingResponse is the response defined by the user for a pending request.
type PendingResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
	Cancel  bool
}

type Server struct {
	mu         sync.RWMutex
	mocks      []mock.Mock
	mux        *router.DynamicMux
	httpServer *http.Server
	MocksDir   string
	Port       int
	ReqLog     *RequestLog

	captureMode bool
	pendingMu     sync.Mutex
	PendingCh     chan PendingRequest
	pendingQueue  []*PendingRequest
}

func New(port int, mocksDir string) *Server {
	mux := router.NewDynamicMux()
	s := &Server{
		mux:       mux,
		MocksDir:  mocksDir,
		Port:      port,
		ReqLog:    NewRequestLog(1000),
		PendingCh: make(chan PendingRequest, 1),
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	mux.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		if !s.CaptureMode() {
			http.NotFound(w, r)
			return
		}

		bodyBytes, _ := io.ReadAll(r.Body)

		pr := &PendingRequest{
			Method:     r.Method,
			Path:       r.URL.Path,
			Headers:    r.Header.Clone(),
			Body:       string(bodyBytes),
			ResponseCh: make(chan PendingResponse, 1),
		}

		s.EnqueuePending(pr)

		select {
		case resp := <-pr.ResponseCh:
			if resp.Cancel {
				http.NotFound(w, r)
				return
			}
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			w.WriteHeader(resp.Status)
			w.Write(resp.Body)
		case <-r.Context().Done():
			// Client disconnected; mock will still be saved if user completes the form
		}
	}

	return s
}

// LoadAndRegister loads mocks from disk and registers them on the mux.
func (s *Server) LoadAndRegister() error {
	mocks, err := mock.LoadMocksFromFS(s.MocksDir)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.mocks = mocks
	s.mu.Unlock()

	s.rebuildRoutes()
	return nil
}

func (s *Server) rebuildRoutes() {
	s.mu.RLock()
	mocks := make([]mock.Mock, len(s.mocks))
	copy(mocks, s.mocks)
	s.mu.RUnlock()

	s.mux.Clear()
	router.RegisterMocks(s.mux, mocks, s.ReqLog)
}

// Start begins serving HTTP in a new goroutine.
func (s *Server) Start() {
	go func() {
		log.Printf("Starting server on :%d", s.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %s", err)
		}
	}()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Mocks() []mock.Mock {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]mock.Mock, len(s.mocks))
	copy(out, s.mocks)
	return out
}

func (s *Server) AddMock(m mock.Mock) {
	s.mu.Lock()
	s.mocks = append(s.mocks, m)
	s.mu.Unlock()
	s.rebuildRoutes()
}

func (s *Server) UpdateMock(index int, m mock.Mock) {
	s.mu.Lock()
	if index >= 0 && index < len(s.mocks) {
		s.mocks[index] = m
	}
	s.mu.Unlock()
	s.rebuildRoutes()
}

func (s *Server) DeleteMock(index int) {
	s.mu.Lock()
	if index >= 0 && index < len(s.mocks) {
		s.mocks = append(s.mocks[:index], s.mocks[index+1:]...)
	}
	s.mu.Unlock()
	s.rebuildRoutes()
}

func (s *Server) ReloadMocks() error {
	return s.LoadAndRegister()
}

func (s *Server) MockCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.mocks)
}

// CaptureMode returns whether capture mode is enabled.
func (s *Server) CaptureMode() bool {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	return s.captureMode
}

// SetCaptureMode toggles capture mode. When turning off, queued requests are cancelled.
func (s *Server) SetCaptureMode(on bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.captureMode = on
	if !on {
		// Flush queued requests with cancel
		for _, pr := range s.pendingQueue {
			select {
			case pr.ResponseCh <- PendingResponse{Cancel: true}:
			default:
			}
		}
		s.pendingQueue = nil
	}
}

// EnqueuePending sends a pending request to the TUI or queues it if the TUI is busy.
func (s *Server) EnqueuePending(pr *PendingRequest) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	select {
	case s.PendingCh <- *pr:
	default:
		s.pendingQueue = append(s.pendingQueue, pr)
	}
}

// NextPending sends the next queued request to PendingCh, if any.
func (s *Server) NextPending() {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if len(s.pendingQueue) > 0 {
		next := s.pendingQueue[0]
		s.pendingQueue = s.pendingQueue[1:]
		select {
		case s.PendingCh <- *next:
		default:
		}
	}
}
