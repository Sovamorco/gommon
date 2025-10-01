package cache

import (
	"context"
	"errors"
)

var ErrNotExist = errors.New("key does not exist")

type Cache interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}
