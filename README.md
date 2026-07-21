# 🛡️ Go Rate Limiter

Rate Limiter em Go desenvolvido como desafio técnico da Pós Go Expert (Full Cycle).

## 📋 Descrição

Middleware HTTP de controle de taxa de acesso com as seguintes características:

- **Limitação por IP**: Limite configurável de requisições por segundo por endereço IP.
- **Limitação por Token (`API_KEY`)**: Limite independente por token no header da requisição.
- **Regra de Ouro (Token > IP)**: Se a requisição contiver o header `API_KEY`, o limite do token tem precedência total sobre o limite do IP.
- **Bloqueio Temporário**: Ao exceder o limite, o cliente é bloqueado por um tempo configurável.
- **Resposta Padronizada**: HTTP `429 Too Many Requests` com body exato:
  ```
  you have reached the maximum number of requests or actions allowed within a certain time frame
  ```
- **Strategy Pattern**: Interface `LimiterStorage` desacoplada — suporta Redis (produção) e Memória (testes).
- **Script Lua Atômico no Redis**: Garante ausência de race conditions sob alta concorrência.

## 🏗️ Arquitetura

```
src/
├── cmd/
│   └── server/
│       └── main.go              # Ponto de entrada (Chi router + Graceful Shutdown)
├── config/
│   └── config.go                # Carregamento de variáveis de ambiente (.env)
├── internal/
│   ├── limiter/
│   │   ├── limiter.go           # RateLimiter — Regra de Ouro (Token > IP)
│   │   ├── limiter_test.go      # Testes: IP, Token, Precedência, Custom Limit
│   │   ├── storage.go           # Interface Strategy: LimiterStorage
│   │   ├── memory_storage.go    # Implementação em memória (thread-safe, para testes)
│   │   └── redis_storage.go     # Implementação Redis (script Lua atômico)
│   └── middleware/
│       ├── rate_limiter.go      # Middleware HTTP: 429 + mensagem exata
│       └── rate_limiter_test.go # Teste de integração do middleware
├── validate/
│   └── main.go                  # Script de validação funcional completa
├── .env.example                 # Template de configuração
├── Dockerfile                   # Multi-stage build (Go 1.23 + Alpine)
├── docker-compose.yaml          # App (8080) + Redis (6379) com healthcheck
├── api.http                     # Coleção de testes com REST Client (VS Code)
├── go.mod
└── go.sum
```

## ⚙️ Configuração

Copie `.env.example` para `.env` e ajuste os valores:

```env
HTTP_PORT=8080

REDIS_HOST=localhost   # Em Docker: use "redis"
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

RATE_LIMIT_IP_MAX_REQUESTS=5             # Máx. req/s por IP
RATE_LIMIT_IP_BLOCK_DURATION_SECONDS=300 # Bloqueio de 5 minutos

RATE_LIMIT_TOKEN_MAX_REQUESTS=10              # Máx. req/s por Token
RATE_LIMIT_TOKEN_BLOCK_DURATION_SECONDS=300   # Bloqueio de 5 minutos
```

## 🚀 Como Executar

### Com Docker (recomendado)

```bash
docker compose up --build -d
```

Isso sobe:
- **`rate_limiter_app`** na porta `8080`
- **`rate_limiter_redis`** na porta `6379` (com healthcheck)

### Localmente (requer Redis rodando)

```bash
go run ./cmd/server/main.go
```

## 🧪 Testes

### Testes automatizados

```bash
go test -v ./...
```

Saída esperada:
```
PASS: TestRateLimiter_IPLimitation
PASS: TestRateLimiter_TokenPrecedenceOverIP
PASS: TestRateLimiter_CustomTokenLimit
PASS: TestRateLimiterMiddleware_AllowedAndBlocked
```

### Validação funcional completa

```bash
go run ./validate/main.go
```

Executa 4 cenários automaticamente: IP limit, Token precedence, bloqueio persistente e validação da mensagem exata.

### Testes manuais com REST Client (VS Code)

Abra `api.http` com a extensão [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) e siga os cenários:

| Cenário | Ação | Resultado Esperado |
|---|---|---|
| 1 — IP limit | Clique "Send Request" 6x no bloco `[IP]` | Req 1–5: `200`, Req 6: `429` |
| 2 — Token limit | Clique 11x no bloco `[TOKEN]` | Req 1–10: `200`, Req 11: `429` |
| 3 — Regra de Ouro | Com IP bloqueado, envie bloco `[OURO]` | `200` (Token sobrepõe bloqueio de IP) |

### Inspecionar o Redis em tempo real

```bash
# Ver chaves de rate/bloqueio criadas
docker exec rate_limiter_redis redis-cli keys "*"

# Monitorar comandos em tempo real
docker exec -it rate_limiter_redis redis-cli monitor

# Resetar todo o estado de rate limiting
docker exec rate_limiter_redis redis-cli FLUSHDB
```

## 🔑 Exemplo de Uso

```bash
# Requisição sem token (limitada por IP)
curl http://localhost:8080/

# Requisição com token (limite independente, maior que IP)
curl -H "API_KEY: meu-token-secreto" http://localhost:8080/

# Resposta ao exceder o limite:
# HTTP 429
# you have reached the maximum number of requests or actions allowed within a certain time frame
```

## 🧠 Decisões de Design

### Strategy Pattern
A interface `LimiterStorage` desacopla completamente a lógica de negócio do mecanismo de persistência. Trocar Redis por outro backend requer apenas uma nova implementação da interface — sem alterar o `RateLimiter` ou o middleware.

### Script Lua Atômico no Redis
A implementação Redis usa um script Lua executado como operação atômica para evitar race conditions. O script verifica bloqueio, incrementa contador, define TTL e ativa bloqueio em uma única operação indivisível.

### Fallback para MemoryStorage
Se o Redis não estiver disponível no boot, o servidor loga um aviso e usa `MemoryStorage`. Isso garante resiliência — a aplicação nunca falha por indisponibilidade do storage.

## 📦 Dependências

| Pacote | Uso |
|---|---|
| `github.com/go-chi/chi/v5` | Router HTTP |
| `github.com/redis/go-redis/v9` | Client Redis |
| `github.com/joho/godotenv` | Carregamento de `.env` |
| `github.com/stretchr/testify` | Assertions nos testes |
