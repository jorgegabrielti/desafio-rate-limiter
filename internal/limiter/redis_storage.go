package limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implementa a estratégia de armazenamento no Redis utilizando Scripting Lua atômico.
type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{
		client: client,
	}
}

// Script Lua executado atomicamente no Redis para evitar condições de corrida (Race Conditions)
var rateLimitLuaScript = redis.NewScript(`
	local blocked_key = KEYS[1]
	local rate_key = KEYS[2]
	local limit = tonumber(ARGV[1])
	local window_ttl = tonumber(ARGV[2])
	local block_ttl = tonumber(ARGV[3])

	-- 1. Verifica se a chave está bloqueada
	if redis.call("EXISTS", blocked_key) == 1 then
		return 0
	end

	-- 2. Incrementa o contador na janela atual
	local current = redis.call("INCR", rate_key)
	if current == 1 then
		redis.call("EXPIRE", rate_key, window_ttl)
	end

	-- 3. Se ultrapassar o limite, define o bloqueio e remove o contador
	if current > limit then
		redis.call("SET", blocked_key, "1", "EX", block_ttl)
		redis.call("DEL", rate_key)
		return 0
	end

	return 1
`)

func (r *RedisStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockedKey := fmt.Sprintf("blocked:%s", key)
	val, err := r.client.Exists(ctx, blockedKey).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return val > 0, nil
}

func (r *RedisStorage) Allow(ctx context.Context, key string, limit int, window time.Duration, blockDuration time.Duration) (bool, error) {
	blockedKey := fmt.Sprintf("blocked:%s", key)
	nowSec := time.Now().Unix()
	rateKey := fmt.Sprintf("rate:%s:%d", key, nowSec)

	windowTTLSec := int(window.Seconds())
	if windowTTLSec < 1 {
		windowTTLSec = 1
	}

	blockTTLSec := int(blockDuration.Seconds())
	if blockTTLSec < 1 {
		blockTTLSec = 1
	}

	keys := []string{blockedKey, rateKey}
	args := []interface{}{
		strconv.Itoa(limit),
		strconv.Itoa(windowTTLSec + 1), // TTL da janela com margem de 1s
		strconv.Itoa(blockTTLSec),
	}

	res, err := rateLimitLuaScript.Run(ctx, r.client, keys, args...).Int()
	if err != nil {
		return false, fmt.Errorf("%w: erro ao executar script no Redis: %v", ErrStorageUnavailable, err)
	}

	return res == 1, nil
}

func (r *RedisStorage) Reset(ctx context.Context, key string) error {
	blockedKey := fmt.Sprintf("blocked:%s", key)
	pattern := fmt.Sprintf("rate:%s:*", key)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, blockedKey)

	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	_, err := pipe.Exec(ctx)
	return err
}
