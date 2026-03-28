package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mic-360/wimo/internal/state"
)

type Logger struct {
	mu      sync.RWMutex
	entries []state.LogEntry
	limit   int
	file    *os.File
}

func NewLogger(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(logDir, time.Now().Format("20060102_150405")+"_winmole.log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{limit: 500, file: file}, nil
}

func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Entries() []state.LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	clone := make([]state.LogEntry, len(l.entries))
	copy(clone, l.entries)
	return clone
}

func (l *Logger) Log(level state.LogLevel, source, message string) {
	entry := state.LogEntry{Time: time.Now(), Level: level, Source: source, Message: message}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.limit {
		l.entries = append([]state.LogEntry{}, l.entries[len(l.entries)-l.limit:]...)
	}
	if l.file != nil {
		fmt.Fprintf(l.file, "[%s] [%s] [%s] %s\n", entry.Time.Format(time.RFC3339), level, source, message)
	}
}

func (l *Logger) Debug(source, message string) { l.Log(state.LogDebug, source, message) }
func (l *Logger) Info(source, message string)  { l.Log(state.LogInfo, source, message) }
func (l *Logger) Warn(source, message string)  { l.Log(state.LogWarn, source, message) }
func (l *Logger) Error(source, message string) { l.Log(state.LogError, source, message) }
