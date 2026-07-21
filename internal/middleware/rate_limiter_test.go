package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jorgegabrielti/go-rate-limiter/config"
	"github.com/jorgegabrielti/go-rate-limiter/internal/limiter"
	"github.com/jorgegabrielti/go-rate-limiter/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiterMiddleware_AllowedAndBlocked(t *testing.T) {
	cfg := &config.Config{
		IPMaxRequests:   2,
		IPBlockDuration: 1 * time.Second,
	}
	storage := limiter.NewMemoryStorage()
	rl := limiter.NewRateLimiter(storage, cfg)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	})

	ts := httptest.NewServer(middleware.RateLimiterMiddleware(rl)(handler))
	defer ts.Close()

	client := ts.Client()

	// 1ª e 2ª requisições -> HTTP 200
	resp1, err := client.Get(ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	resp2, err := client.Get(ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// 3ª requisição -> HTTP 429 com a mensagem exata requerida no enunciado
	resp3, err := client.Get(ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, resp3.StatusCode)

	buf := make([]byte, 2048)
	n, _ := resp3.Body.Read(buf)
	bodyStr := string(buf[:n])

	expectedBody := "you have reached the maximum number of requests or actions allowed within a certain time frame"
	assert.Equal(t, expectedBody, bodyStr)
}
