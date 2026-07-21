package limiter

import (
	"context"
	"errors"
	"time"
)

var ErrStorageUnavailable = errors.New("storage backend unavailable")

type LimiterStorage interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration, blockDuration time.Duration) (allowed bool, err error)
	IsBlocked(ctx context.Context, key string) (blocked bool, err error)
	Reset(ctx context.Context, key string) error
}
