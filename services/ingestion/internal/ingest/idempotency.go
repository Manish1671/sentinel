package ingest

import (
	"sync"
	"time"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/apierr"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/events"
)

type record struct {
	hash      string
	envelope  events.Envelope
	expiresAt time.Time
}

// Memory is a process-local idempotency index. It is not durable.
// Keys are Idempotency-Key / idempotency_key, or event_id when those are absent.
// Downstream consumers must still deduplicate by event_id.
type Memory struct {
	mu      sync.Mutex
	items   map[string]record
	maxSize int
	ttl     time.Duration
	now     func() time.Time
}

func NewMemory() *Memory {
	return &Memory{
		items:   map[string]record{},
		maxSize: 10000,
		ttl:     24 * time.Hour,
		now:     time.Now,
	}
}

func (m *Memory) Check(key, hash string) (events.Envelope, bool, error) {
	if key == "" {
		return events.Envelope{}, false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evictLocked()
	rec, ok := m.items[key]
	if !ok {
		return events.Envelope{}, false, nil
	}
	if rec.hash != hash {
		return events.Envelope{}, false, apierr.IdempotencyConflict()
	}
	return rec.envelope, true, nil
}

func (m *Memory) Remember(key, hash string, env events.Envelope) {
	if key == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evictLocked()
	if len(m.items) >= m.maxSize {
		for k := range m.items {
			delete(m.items, k)
			break
		}
	}
	m.items[key] = record{hash: hash, envelope: env, expiresAt: m.now().Add(m.ttl)}
}

func (m *Memory) evictLocked() {
	now := m.now()
	for k, rec := range m.items {
		if now.After(rec.expiresAt) {
			delete(m.items, k)
		}
	}
}
