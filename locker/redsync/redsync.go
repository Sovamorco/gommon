package redlock

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/locker"
)

const (
	ExpiryTime = 15 * time.Second
)

type Redsync struct {
	rs     *redsync.Redsync
	prefix string
}

func New(rs *redsync.Redsync, prefix string) *Redsync {
	return &Redsync{
		rs:     rs,
		prefix: prefix,
	}
}

// interface return required by interface.
//
//nolint:ireturn
func (rl *Redsync) Lock(ctx context.Context, name string) (locker.Lock, error) {
	mutex := rl.rs.NewMutex(rl.prefix+":"+name, redsync.WithExpiry(ExpiryTime))

	err := mutex.TryLockContext(ctx)
	if err != nil {
		return nil, errorx.Decorate(err, "lock mutex")
	}

	l := &Lock{
		mutex: mutex,
		stop:  make(chan struct{}),
	}

	l.startExtension(ctx)

	return l, nil
}
