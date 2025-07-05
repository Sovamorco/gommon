package mock

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rs/zerolog"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/broker"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	broker.Register("mock", newMock)
}

type Broker struct {
	handlers map[string][]broker.MessageHandler
	mu       sync.RWMutex `exhaustruct:"optional"`
}

//nolint:ireturn // required by broker.Register.
func newMock(_ context.Context, _ string) (broker.Broker, error) {
	return &Broker{
		handlers: make(map[string][]broker.MessageHandler),
	}, nil
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

func (b *Broker) Publish(ctx context.Context, channel string, payload any) error {
	bs, err := json.Marshal(payload)
	if err != nil {
		return errorx.Wrap(err, "marshal payload")
	}

	logger := zerolog.Ctx(ctx)

	logger.Debug().
		Str("channel", channel).Str("payload", string(bs)).
		Msg("Mock broker publish")

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, mh := range b.handlers[channel] {
		go func() {
			err := mh(ctx, channel, bs)
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
