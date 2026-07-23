package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jorgegabrielti/go-rate-limiter/config"
	"github.com/jorgegabrielti/go-rate-limiter/internal/limiter"
	customMiddleware "github.com/jorgegabrielti/go-rate-limiter/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	log.Printf("🚀 Inicializando Rate Limiter Server na porta :%s...", cfg.HTTPPort)
	log.Printf("📊 IP Max=%d req/s (Bloqueio %s) | Token Max=%d req/s (Bloqueio %s)",
		cfg.IPMaxRequests, cfg.IPBlockDuration, cfg.TokenMaxRequests, cfg.TokenBlockDuration)

	var storage limiter.LimiterStorage

	redisAddr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctxPing, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelPing()

	if err := rdb.Ping(ctxPing).Err(); err != nil {
		log.Printf("⚠️ Redis indisponível (%s): %v. Usando MemoryStorage como fallback.", redisAddr, err)
		storage = limiter.NewMemoryStorage()
	} else {
		log.Printf("✅ Conectado ao Redis (%s)", redisAddr)
		storage = limiter.NewRedisStorage(rdb)
	}

	rateLimiter := limiter.NewRateLimiter(storage, cfg)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.RateLimiterMiddleware(rateLimiter))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Bem-vindo ao serviço protegido pelo Rate Limiter em Go!"}`))
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "pong"}`))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro no servidor HTTP: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Encerrando servidor...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Erro no encerramento: %v", err)
	}

	log.Println("👋 Servidor finalizado.")
}
