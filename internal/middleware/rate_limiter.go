package middleware

import (
	"net/http"

	"github.com/jorgegabrielti/go-rate-limiter/internal/limiter"
)

const BlockedMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"

// RateLimiterMiddleware cria um middleware HTTP compatível com net/http e routers como chi.
func RateLimiterMiddleware(l *limiter.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := l.AllowRequest(r.Context(), r)
			if err != nil {
				// Em caso de erro na camada de infraestrutura/storage, permite a requisição por resiliência
				// ou pode registrar log de alerta.
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests) // 429
				_, _ = w.Write([]byte(BlockedMessage))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
