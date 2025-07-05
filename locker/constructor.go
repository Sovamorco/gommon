package locker

import (
	"context"
	"fmt"
	"sync"
)

type UnknownProviderError struct {
	Provider string
}

func (e UnknownProviderError) Error() string {
	return fmt.Sprintf("locker: unknown provider %q", e.Provider)
}

type builder func(ctx context.Context, connst string) (Locker, error)

//nolint:gochecknoglobals // driver pattern.
var (
	providersMu sync.RWMutex
	providers   = make(map[string]builder)
)

func Register(p string, bf builder) {
	providersMu.Lock()
	defer providersMu.Unlock()

	if _, ok := providers[p]; ok {
		panic("locker: provider already registered: " + p)
	}

	providers[p] = bf
}

//nolint:ireturn // depends on provider.
func New(ctx context.Context, provider, connst string) (Locker, error) {
	providersMu.RLock()

	pf, ok := providers[provider]

	providersMu.RUnlock()

	if !ok {
		return nil, UnknownProviderError{
			Provider: provider,
		}
	}

	return pf(ctx, connst)
}
