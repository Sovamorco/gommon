package mock

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/sovamorco/gommon/broker"
)

var _ = (broker.Broker)((*Broker)(nil))

type Broker struct {
	handlers map[string][]broker.MessageHandler
	mu       sync.RWMutex `exhaustruct:"optional"`
}

func New() *Broker {
	return &Broker{
		handlers: make(map[string][]broker.MessageHandler),
	}
}

func (b *Broker) Subscribe(ctx context.Context, mh broker.MessageHandler, channels ...string) {
	zerolog.Ctx(ctx).Debug().Strs("channels", channels).Msg("Mock broker subscribe")

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, c := range channels {
		if _, ok := b.handlers[c]; !ok {
			b.handlers[c] = make([]broker.MessageHandler, 0, 1)
		}

		b.handlers[c] = append(b.handlers[c], mh)
	}
}

func (b *Broker) Publish(ctx context.Context, channel, payload string) error {
	logger := zerolog.Ctx(ctx)

	logger.Debug().
		Str("channel", channel).Str("payload", payload).
		Msg("Mock broker publish")

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, mh := range b.handlers[channel] {
		go func() {
			err := mh(ctx, channel, payload)
			if err != nil {
				logger.Error().Err(err).Msg("Mock broker failed to process message")
			}
		}()
	}

	return nil
}

func (b *Broker) Shutdown(ctx context.Context) {
	zerolog.Ctx(ctx).Debug().Msg("Mock broker shutdown")
}
