package mock

import (
	"context"

	"github.com/sovamorco/gommon/cache"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	cache.Register("mock", newMock)
}

type Cache struct {
	m map[string][]byte
}

//nolint:ireturn // required by cache.Register.
func newMock(_ context.Context, _ string) (cache.Cache, error) {
	return &Cache{
		m: make(map[string][]byte),
	}, nil
}

func (c *Cache) Set(_ context.Context, key string, value []byte) error {
	c.m[key] = value

	return nil
}

func (c *Cache) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := c.m[key]
	if !ok {
		return nil, cache.ErrNotExist
	}

	return v, nil
}
