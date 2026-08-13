package cache

import (
	"context"
	"time"
)

// Cache defines the minimal key-value operations the application depends on.
// UpstashClient implements this interface.
// Any other cache backend (native Redis, in-memory, DynamoDB) can be swapped
// in by implementing these five methods — without changing any business logic.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
}
