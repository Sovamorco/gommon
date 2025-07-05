package redlock

import (
	"context"
	"net/url"
	"time"

	"github.com/go-redsync/redsync/v4"
	rsredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/gredis"
	"github.com/sovamorco/gommon/locker"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	locker.Register("redis", newRedsync)
}

const (
	ExpiryTime = 15 * time.Second
)

type Redsync struct {
	rs     *redsync.Redsync
	prefix string
}

//nolint:ireturn // required by locker.Register.
func newRedsync(ctx context.Context, connst string) (locker.Locker, error) {
	u, err := url.Parse(connst)
	if err != nil {
		return nil, errorx.Wrap(err, "parse connection url")
	}

	cl, err := gredis.New(ctx, connst)
	if err != nil {
		return nil, errorx.Wrap(err, "create redis client")
	}

	rsp := rsredis.NewPool(cl)

	rs := redsync.New(rsp)

	return &Redsync{
		rs:     rs,
		prefix: u.Fragment,
	}, nil
}

// interface return required by interface.
//
//nolint:ireturn
func (rl *Redsync) Lock(ctx context.Context, name string) (locker.Lock, error) {
	mutex := rl.rs.NewMutex(rl.prefix+":"+name, redsync.WithExpiry(ExpiryTime))

	err := mutex.TryLockContext(ctx)
	if err != nil {
		return nil, errorx.Wrap(err, "lock mutex")
	}

	l := &Lock{
		mutex: mutex,
		stop:  make(chan struct{}),
	}

	l.startExtension(ctx)

	return l, nil
}
