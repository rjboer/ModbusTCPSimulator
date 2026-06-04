package modbus

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type MemoryLogSink struct {
	mu      sync.Mutex
	entries []string
}

func NewMemoryLogSink() *MemoryLogSink {
	return &MemoryLogSink{}
}

func (s *MemoryLogSink) Write(p []byte) (int, error) {
	entry := strings.TrimRight(string(p), "\r\n")
	if entry == "" {
		return len(p), nil
	}

	s.mu.Lock()
	s.entries = append(s.entries, entry)
	s.mu.Unlock()
	return len(p), nil
}

func (s *MemoryLogSink) Snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *MemoryLogSink) SnapshotSince(offset int) ([]string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if offset < 0 {
		offset = 0
	}
	if offset > len(s.entries) {
		offset = len(s.entries)
	}

	out := make([]string, len(s.entries)-offset)
	copy(out, s.entries[offset:])
	return out, len(s.entries)
}

func (s *MemoryLogSink) Export(path string) error {
	lines := s.Snapshot()
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func DefaultLogExportPath() string {
	return fmt.Sprintf("mock-modbus-log-%s.log", time.Now().Format("20060102-150405"))
}
