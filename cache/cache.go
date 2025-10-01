package cache

import (
	"context"
	"errors"
	"time"
)

var ErrNotExist = errors.New("key does not exist")

type Cache interface {
	Set(ctx context.Context, key string, value []byte, lifetime time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
}
