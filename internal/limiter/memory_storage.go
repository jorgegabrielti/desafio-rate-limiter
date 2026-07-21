package limiter

import (
	"context"
	"sync"
	"time"
)

type MemoryStorage struct {
	mu         sync.RWMutex
	blocked    map[string]time.Time
	timestamps map[string][]time.Time
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		blocked:    make(map[string]time.Time),
		timestamps: make(map[string][]time.Time),
	}
}

func (m *MemoryStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blockUntil, exists := m.blocked[key]
	if !exists {
		return false, nil
	}

	return time.Now().Before(blockUntil), nil
}

func (m *MemoryStorage) Allow(ctx context.Context, key string, limit int, window time.Duration, blockDuration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	if blockUntil, exists := m.blocked[key]; exists {
		if now.Before(blockUntil) {
			return false, nil
		}
		delete(m.blocked, key)
	}

	cutoff := now.Add(-window)
	var recent []time.Time
	for _, t := range m.timestamps[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= limit {
		m.blocked[key] = now.Add(blockDuration)
		m.timestamps[key] = nil
		return false, nil
	}

	recent = append(recent, now)
	m.timestamps[key] = recent
	return true, nil
}

func (m *MemoryStorage) Reset(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.blocked, key)
	delete(m.timestamps, key)
	return nil
}
