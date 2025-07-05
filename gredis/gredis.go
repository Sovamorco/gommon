package gredis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/sovamorco/errorx"
)

func New(ctx context.Context, connurl string) (*redis.Client, error) {
	opts, err := redis.ParseURL(connurl)
	if err != nil {
		return nil, errorx.Wrap(err, "parse connection url")
	}

	rsc := redis.NewClient(
		opts,
	)

	err = rsc.Ping(ctx).Err()
	if err != nil {
		return nil, errorx.Wrap(err, "ping redis")
	}

	return rsc, nil
}
