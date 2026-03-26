package server

import (
	"sync"
	"time"

	"simple-mock-server/internal/router"
)

type LogEntry struct {
	Timestamp    time.Time
	Method       string
	Path         string
	Status       int
	RequestBody  string
	ResponseTime int
}

// RequestLog is a thread-safe ring buffer that also broadcasts new entries
// to subscribers via channels.
type RequestLog struct {
	mu          sync.Mutex
	entries     []LogEntry
	maxSize     int
	subscribers []chan LogEntry
}

func NewRequestLog(maxSize int) *RequestLog {
	return &RequestLog{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Log implements router.RequestLogger.
func (rl *RequestLog) Log(entry router.RequestEntry) {
	le := LogEntry{
		Timestamp:    time.Now(),
		Method:       entry.Method,
		Path:         entry.Path,
		Status:       entry.Status,
		RequestBody:  entry.RequestBody,
		ResponseTime: entry.ResponseTime,
	}

	rl.mu.Lock()
	if len(rl.entries) >= rl.maxSize {
		rl.entries = rl.entries[1:]
	}
	rl.entries = append(rl.entries, le)
	subs := make([]chan LogEntry, len(rl.subscribers))
	copy(subs, rl.subscribers)
	rl.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- le:
		default:
			// drop if subscriber is slow
		}
	}
}

func (rl *RequestLog) Entries() []LogEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	out := make([]LogEntry, len(rl.entries))
	copy(out, rl.entries)
	return out
}

func (rl *RequestLog) Clear() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.entries = rl.entries[:0]
}

func (rl *RequestLog) Subscribe() chan LogEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	ch := make(chan LogEntry, 100)
	rl.subscribers = append(rl.subscribers, ch)
	return ch
}

func (rl *RequestLog) Unsubscribe(ch chan LogEntry) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for i, sub := range rl.subscribers {
		if sub == ch {
			rl.subscribers = append(rl.subscribers[:i], rl.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}
