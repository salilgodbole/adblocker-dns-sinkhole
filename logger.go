package main

import (
	"sync"
	"time"
)

type LogStatus string

const (
	StatusAllowed        LogStatus = "Allowed"
	StatusBlockedCustom  LogStatus = "Blocked (Custom)"
	StatusBlockedDefault LogStatus = "Blocked (Community)"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	ClientIP  string    `json:"client_ip"`
	Domain    string    `json:"domain"`
	Status    LogStatus `json:"status"`
	QueryType string    `json:"query_type,omitempty"`
	Response  string    `json:"response,omitempty"`
}

type QueryLogger struct {
	mu       sync.RWMutex
	entries  []LogEntry
	maxSize  int
	tracking bool
}

func NewQueryLogger(maxSize int) *QueryLogger {
	return &QueryLogger{
		entries:  make([]LogEntry, 0, maxSize),
		maxSize:  maxSize,
		tracking: false,
	}
}

func (l *QueryLogger) IsTrackingEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.tracking
}

func (l *QueryLogger) SetTracking(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tracking = enabled
}

func (l *QueryLogger) Log(clientIP, domain string, status LogStatus, queryType string, response string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		ClientIP:  clientIP,
		Domain:    domain,
		Status:    status,
		QueryType: queryType,
		Response:  response,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, entry)
	if len(l.entries) > l.maxSize {
		// Remove oldest element (ring buffer behavior)
		l.entries = l.entries[1:]
	}
}

func (l *QueryLogger) GetLogs() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to avoid race conditions
	logs := make([]LogEntry, len(l.entries))
	// Copy in reverse order so newest is first
	for i, j := 0, len(l.entries)-1; i < len(l.entries); i, j = i+1, j-1 {
		logs[i] = l.entries[j]
	}
	return logs
}
