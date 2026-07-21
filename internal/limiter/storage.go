package limiter

import (
	"context"
	"errors"
	"time"
)

var ErrStorageUnavailable = errors.New("storage backend unavailable")

// LimiterStorage define o contrato para o padrão Strategy de persistência do Rate Limiter.
type LimiterStorage interface {
	// Allow verifica se a chave (IP ou Token) pode realizar a requisição com base no limite por segundo.
	// Retorna allowed = true se permitida, ou allowed = false se bloqueada/excedida.
	Allow(ctx context.Context, key string, limit int, window time.Duration, blockDuration time.Duration) (allowed bool, err error)

	// IsBlocked verifica se uma determinada chave está ativamente bloqueada.
	IsBlocked(ctx context.Context, key string) (blocked bool, err error)

	// Reset limpa contadores/bloqueios de uma chave (útil para testes).
	Reset(ctx context.Context, key string) error
}
