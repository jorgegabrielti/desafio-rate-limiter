package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/jorgegabrielti/go-rate-limiter/config"
	"github.com/jorgegabrielti/go-rate-limiter/internal/limiter"
	"github.com/jorgegabrielti/go-rate-limiter/internal/middleware"
)

func main() {
	fmt.Println("=======================================================")
	fmt.Println("  VALIDAÇÃO FUNCIONAL COMPLETA - Desafio Rate Limiter  ")
	fmt.Println("=======================================================")

	cfg := &config.Config{
		IPMaxRequests:             3,
		IPBlockDuration:           5 * time.Second,
		TokenMaxRequests:          6,
		TokenBlockDuration:        5 * time.Second,
		CustomTokenLimits:         make(map[string]int),
		CustomTokenBlockDurations: make(map[string]time.Duration),
	}

	storage := limiter.NewMemoryStorage()
	rl := limiter.NewRateLimiter(storage, cfg)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	ts := httptest.NewServer(middleware.RateLimiterMiddleware(rl)(okHandler))
	defer ts.Close()
	client := ts.Client()

	// ----------------------------------------------------------------
	fmt.Println("\n[CENÁRIO 1] Limitação por IP (limite: 3 req/s)")
	fmt.Println("----------------------------------------------------------------")
	for i := 1; i <= 5; i++ {
		resp, _ := client.Get(ts.URL + "/")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			fmt.Printf("  Req #%d | Status: %d 🚫 | Body: %q\n", i, resp.StatusCode, string(body))
		} else {
			fmt.Printf("  Req #%d | Status: %d ✅\n", i, resp.StatusCode)
		}
	}

	// ----------------------------------------------------------------
	// Reseta o IP para o próximo cenário
	_ = storage.Reset(context.Background(), "ip:127.0.0.1")

	fmt.Println("\n[CENÁRIO 2] Regra de Ouro — Precedência Token > IP")
	fmt.Println("(IP com limite 3 req/s, mas Token tem limite 6 req/s)")
	fmt.Println("----------------------------------------------------------------")
	for i := 1; i <= 8; i++ {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
		req.Header.Set("API_KEY", "meu-token-valido")
		resp, _ := client.Do(req)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			fmt.Printf("  Req #%d | Status: %d 🚫 | Body: %q\n", i, resp.StatusCode, string(body))
		} else {
			fmt.Printf("  Req #%d | Status: %d ✅\n", i, resp.StatusCode)
		}
	}

	// ----------------------------------------------------------------
	fmt.Println("\n[CENÁRIO 3] Bloqueio permanece ativo após exceder limite")
	fmt.Println("(Requisições subsequentes com mesmo IP após bloqueio)")
	fmt.Println("----------------------------------------------------------------")
	for i := 1; i <= 3; i++ {
		resp, _ := client.Get(ts.URL + "/")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("  Req #%d pós-bloqueio | Status: %d | Body: %q\n", i, resp.StatusCode, string(body))
	}

	// ----------------------------------------------------------------
	fmt.Println("\n[CENÁRIO 4] Valida a mensagem exata exigida pelo enunciado")
	fmt.Println("----------------------------------------------------------------")
	expectedMsg := "you have reached the maximum number of requests or actions allowed within a certain time frame"
	resp, _ := client.Get(ts.URL + "/")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	actualMsg := string(body)
	if actualMsg == expectedMsg {
		fmt.Println("  ✅ PASS: Mensagem de bloqueio está CORRETA!")
	} else {
		fmt.Printf("  ❌ FAIL: Mensagem incorreta!\n  Esperado: %q\n  Recebido: %q\n", expectedMsg, actualMsg)
	}

	fmt.Println("\n=======================================================")
	fmt.Println("          VALIDAÇÃO CONCLUÍDA COM SUCESSO! 🎉           ")
	fmt.Println("=======================================================")
}
