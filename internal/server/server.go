package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/router"
)

// handleCapture blocks the HTTP handler until the TUI user defines a response.
// It is used both by the NotFoundHandler and when capture mode intercepts a matched route.
func (s *Server) handleCapture(w http.ResponseWriter, r *http.Request) {
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

// SelectionRequest is sent to the TUI when multiple mocks match a request.
type SelectionRequest struct {
	Path        string
	Verb        string
	Mocks       []mock.Mock
	CaptureMode bool // true when capture mode is on; picker shows "Define new" option
	ResponseCh  chan int // index of selected mock, -1 to cancel, -2 to define new
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

	SelectionCh chan SelectionRequest
}

func New(port int, mocksDir string) *Server {
	mux := router.NewDynamicMux()
	s := &Server{
		mux:         mux,
		MocksDir:    mocksDir,
		Port:        port,
		ReqLog:      NewRequestLog(1000),
		PendingCh:   make(chan PendingRequest, 1),
		SelectionCh: make(chan SelectionRequest, 1),
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	mux.NotFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		if !s.CaptureMode() {
			s.ReqLog.Log(router.RequestEntry{
				Method: r.Method,
				Path:   r.RequestURI,
				Status: http.StatusNotFound,
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error":  "no mock defined for this route",
				"method": r.Method,
				"path":   r.URL.Path,
			})
			return
		}
		s.handleCapture(w, r)
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
	router.RegisterMocks(s.mux, mocks, router.RouteConfig{
		Logger:       s.ReqLog,
		Selector:     s.selectMock,
		CaptureCheck: s.CaptureMode,
		Fallback:     s.handleCapture,
	})
}

// selectMock is the MockSelector used in TUI mode. It sends a SelectionRequest
// to the TUI and blocks until the user picks a mock.
func (s *Server) selectMock(path string, verb string, mocks []mock.Mock) int {
	req := SelectionRequest{
		Path:        path,
		Verb:        verb,
		Mocks:       mocks,
		CaptureMode: s.CaptureMode(),
		ResponseCh:  make(chan int, 1),
	}
	s.SelectionCh <- req
	return <-req.ResponseCh
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

func (s *Server) ToggleMock(index int) {
	s.mu.Lock()
	if index >= 0 && index < len(s.mocks) {
		s.mocks[index].Disabled = !s.mocks[index].Disabled
		mock.SaveMockToFS(s.MocksDir, s.mocks[index])
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
	n := 0
	for _, m := range s.mocks {
		if !m.Disabled {
			n++
		}
	}
	return n
}

func (s *Server) TotalMockCount() int {
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
