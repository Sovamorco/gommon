package locker

import (
	"context"

	"github.com/rs/zerolog"
)

type Locker interface {
	Lock(ctx context.Context, name string) (Lock, error)
}

type Lock interface {
	Unlock(ctx context.Context) error
}

// useful for defers.
func UnlockLog(ctx context.Context, l Lock) {
	logger := zerolog.Ctx(ctx)

	err := l.Unlock(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to unlock lock")
	}
}
