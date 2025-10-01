package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sovamorco/gommon/cache"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	cache.Register("mock", newMock)
}

type cacheValue struct {
	versionKey string
	content    []byte
}

type Cache struct {
	m map[string]cacheValue
}

//nolint:ireturn // required by cache.Register.
func newMock(_ context.Context, _ string) (cache.Cache, error) {
	return &Cache{
		m: make(map[string]cacheValue),
	}, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, lifetime time.Duration) error {
	vk := uuid.New().String()

	c.m[key] = cacheValue{
		versionKey: vk,
		content:    value,
	}

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(lifetime):
		}

		v, ok := c.m[key]
		if !ok {
			return
		}

		if v.versionKey == vk {
			delete(c.m, key)
		}
	}()

	return nil
}

func (c *Cache) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := c.m[key]
	if !ok {
		return nil, cache.ErrNotExist
	}

	return v.content, nil
}
