package limiter

import (
	"context"
	"sync"
	"time"
)

// MemoryStorage implementa LimiterStorage em memória (thread-safe) para desenvolvimento e testes.
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

	if time.Now().Before(blockUntil) {
		return true, nil
	}

	return false, nil
}

func (m *MemoryStorage) Allow(ctx context.Context, key string, limit int, window time.Duration, blockDuration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// 1. Verifica se a chave está bloqueada
	if blockUntil, exists := m.blocked[key]; exists {
		if now.Before(blockUntil) {
			return false, nil
		}
		// Expira o bloqueio
		delete(m.blocked, key)
	}

	// 2. Filtra requisições passadas mantendo apenas as dentro da janela de tempo (ex: 1 segundo)
	cutoff := now.Add(-window)
	var recent []time.Time
	for _, t := range m.timestamps[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	// 3. Verifica se a adição dessa nova requisição excede o limite
	if len(recent) >= limit {
		// Bloqueia a chave por blockDuration
		m.blocked[key] = now.Add(blockDuration)
		m.timestamps[key] = nil
		return false, nil
	}

	// 4. Registra a requisição atual
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
