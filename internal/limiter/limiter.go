package limiter

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jorgegabrielti/go-rate-limiter/config"
)

type RateLimiter struct {
	storage LimiterStorage
	cfg     *config.Config
}

func NewRateLimiter(storage LimiterStorage, cfg *config.Config) *RateLimiter {
	return &RateLimiter{
		storage: storage,
		cfg:     cfg,
	}
}

func (rl *RateLimiter) AllowRequest(ctx context.Context, r *http.Request) (bool, error) {
	token := strings.TrimSpace(r.Header.Get("API_KEY"))

	if token != "" {
		key := "token:" + token
		limit := rl.cfg.TokenMaxRequests
		if customLimit, exists := rl.cfg.CustomTokenLimits[token]; exists {
			limit = customLimit
		}

		blockDuration := rl.cfg.TokenBlockDuration
		if customDuration, exists := rl.cfg.CustomTokenBlockDurations[token]; exists {
			blockDuration = customDuration
		}

		return rl.storage.Allow(ctx, key, limit, 1*time.Second, blockDuration)
	}

	ip := getClientIP(r)
	key := "ip:" + ip
	limit := rl.cfg.IPMaxRequests
	blockDuration := rl.cfg.IPBlockDuration

	return rl.storage.Allow(ctx, key, limit, 1*time.Second, blockDuration)
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return strings.TrimSpace(realIP)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
