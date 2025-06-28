package redis

import (
	"context"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/broker"
)

var _ = (broker.Broker)((*Broker)(nil))

type Broker struct {
	cl     *redis.Client
	prefix string
}

func New(cl *redis.Client, prefix string) *Broker {
	return &Broker{
		cl:     cl,
		prefix: prefix,
	}
}

func (b *Broker) Subscribe(ctx context.Context, mh broker.MessageHandler, channels ...string) {
	// add prefix for non-system channels.
	for i, c := range channels {
		if !strings.HasPrefix(c, "__") {
			channels[i] = b.prefix + ":" + c
		}
	}

	ps := b.cl.Subscribe(ctx, channels...)

	go b.subscriptionHandler(ctx, mh, ps.Channel())
}

func (b *Broker) Publish(ctx context.Context, channel, message string) error {
	channel = b.prefix + ":" + channel

	err := b.cl.Publish(ctx, channel, []byte(message)).Err()

	return errorx.Wrap(err, "publish")
}

func (b *Broker) Shutdown(ctx context.Context) {
	err := b.cl.Close()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Failed to close redis client")
	}
}

func (b *Broker) subscriptionHandler(ctx context.Context, mh broker.MessageHandler, ch <-chan *redis.Message) {
	logger := zerolog.Ctx(ctx)

	var wg sync.WaitGroup

	var msg *redis.Message

	var ok bool

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case msg, ok = <-ch:
			if !ok {
				break loop
			}
		}

		logger := logger.With().Str("channel", msg.Channel).Logger()
		ctx := logger.WithContext(ctx)

		// ignore messages from keyevent del channel
		if msg.Channel == "__keyevent@0__:del" {
			logger = logger.Level(zerolog.Disabled)
		}

		wg.Add(1)

		go func() {
			defer wg.Done()

			logger.Debug().Str("payload", msg.Payload).Msg("Received message")

			err := mh(ctx, msg.Channel, msg.Payload)
			if err != nil {
				logger.Error().Err(err).Msg("Error processing message")
			}
		}()
	}

	wg.Wait()
}
