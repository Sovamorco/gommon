package redlock

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/rs/zerolog"
	"github.com/sovamorco/errorx"
)

const (
	extendBuffer = 5 * time.Second
)

type Lock struct {
	mutex *redsync.Mutex
	stop  chan struct{}
}

func (l *Lock) Unlock(ctx context.Context) error {
	close(l.stop)

	_, err := l.mutex.UnlockContext(ctx)
	if err != nil {
		return errorx.Decorate(err, "unlock mutex")
	}

	return nil
}

func (l *Lock) startExtension(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	timeRemaining := time.Until(l.mutex.Until())

	t := time.NewTimer(timeRemaining - extendBuffer)

	go func() {
		for {
			select {
			case <-l.stop:
				return
			case <-ctx.Done():
				return
			case <-t.C:
				_, err := l.mutex.ExtendContext(ctx)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to extend mutex")
				}

				timeRemaining = time.Until(l.mutex.Until())
				t.Reset(timeRemaining - extendBuffer)
			}
		}
	}()
}
