package limiter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jorgegabrielti/go-rate-limiter/config"
	"github.com/jorgegabrielti/go-rate-limiter/internal/limiter"
	"github.com/stretchr/testify/assert"
)

func createTestConfig() *config.Config {
	return &config.Config{
		IPMaxRequests:             2, // Limite baixo para IP (2 req/s)
		IPBlockDuration:           1 * time.Second,
		TokenMaxRequests:          5, // Limite mais alto para Token (5 req/s)
		TokenBlockDuration:        1 * time.Second,
		CustomTokenLimits:         make(map[string]int),
		CustomTokenBlockDurations: make(map[string]time.Duration),
	}
}

func TestRateLimiter_IPLimitation(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	storage := limiter.NewMemoryStorage()
	rateLimiter := limiter.NewRateLimiter(storage, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Requisições 1 e 2 devem ser permitidas
	allowed1, err := rateLimiter.AllowRequest(ctx, req)
	assert.NoError(t, err)
	assert.True(t, allowed1)

	allowed2, err := rateLimiter.AllowRequest(ctx, req)
	assert.NoError(t, err)
	assert.True(t, allowed2)

	// Requisição 3 deve exceder o limite e ser bloqueada
	allowed3, err := rateLimiter.AllowRequest(ctx, req)
	assert.NoError(t, err)
	assert.False(t, allowed3, "Deveria bloquear requisições acima do limite de IP (2 req/s)")
}

func TestRateLimiter_TokenPrecedenceOverIP(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	storage := limiter.NewMemoryStorage()
	rateLimiter := limiter.NewRateLimiter(storage, cfg)

	// Simula requisição com IP e com Token no Header API_KEY
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("API_KEY", "secret-token-abc")

	// Como o IP tem limite de 2 req/s, mas o Token tem limite de 5 req/s,
	// devemos conseguir fazer até 5 requisições com sucesso!
	for i := 1; i <= 5; i++ {
		allowed, err := rateLimiter.AllowRequest(ctx, req)
		assert.NoError(t, err)
		assert.True(t, allowed, "Requisição %d com Token válido deveria ser permitida", i)
	}

	// A 6ª requisição deve ser bloqueada pelo limite do Token
	allowed6, err := rateLimiter.AllowRequest(ctx, req)
	assert.NoError(t, err)
	assert.False(t, allowed6, "A 6ª requisição deveria ser bloqueada pois o limite do Token é 5 req/s")
}

func TestRateLimiter_CustomTokenLimit(t *testing.T) {
	ctx := context.Background()
	cfg := createTestConfig()
	cfg.CustomTokenLimits["vip-token"] = 10

	storage := limiter.NewMemoryStorage()
	rateLimiter := limiter.NewRateLimiter(storage, cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("API_KEY", "vip-token")

	// Deve permitir 10 requisições para o token VIP personalizado
	for i := 1; i <= 10; i++ {
		allowed, err := rateLimiter.AllowRequest(ctx, req)
		assert.NoError(t, err)
		assert.True(t, allowed, "Requisição VIP %d deveria ser permitida", i)
	}

	allowed11, err := rateLimiter.AllowRequest(ctx, req)
	assert.NoError(t, err)
	assert.False(t, allowed11, "A 11ª requisição do token VIP deveria ser bloqueada")
}
