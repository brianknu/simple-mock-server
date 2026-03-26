package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/router"
)

type Server struct {
	mu         sync.RWMutex
	mocks      []mock.Mock
	mux        *router.DynamicMux
	httpServer *http.Server
	MocksDir   string
	Port       int
	ReqLog     *RequestLog
}

func New(port int, mocksDir string) *Server {
	mux := router.NewDynamicMux()
	return &Server{
		mux:      mux,
		MocksDir: mocksDir,
		Port:     port,
		ReqLog:   NewRequestLog(1000),
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
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
