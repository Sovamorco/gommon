package redis

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/cache"
	"github.com/sovamorco/gommon/gredis"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	cache.Register("redis", newRedis)
}

type Cache struct {
	c      *redis.Client
	prefix string
}

//nolint:ireturn // required by cache.Register.
func newRedis(ctx context.Context, connurl string) (cache.Cache, error) {
	u, err := url.Parse(connurl)
	if err != nil {
		return nil, errorx.Wrap(err, "parse url")
	}

	c, err := gredis.New(ctx, connurl)
	if err != nil {
		return nil, errorx.Wrap(err, "create redis client")
	}

	err = c.Ping(ctx).Err()

	return &Cache{
		c:      c,
		prefix: u.Fragment,
	}, errorx.Wrap(err, "send ping")
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, lifetime time.Duration) error {
	err := c.c.Set(ctx, c.prefix+":"+key, value, lifetime).Err()

	return errorx.Wrap(err, "set value")
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.c.Get(ctx, c.prefix+":"+key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, cache.ErrNotExist
		}

		return nil, errorx.Wrap(err, "get value")
	}

	return []byte(val), nil
}
