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
}

type QueryLogger struct {
	mu      sync.RWMutex
	entries []LogEntry
	maxSize int
}

func NewQueryLogger(maxSize int) *QueryLogger {
	return &QueryLogger{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

func (l *QueryLogger) Log(clientIP, domain string, status LogStatus) {
	entry := LogEntry{
		Timestamp: time.Now(),
		ClientIP:  clientIP,
		Domain:    domain,
		Status:    status,
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
