package redis

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/broker"
	"github.com/sovamorco/gommon/gredis"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	broker.Register("redis", newRedis)
}

type Broker struct {
	cl     *redis.Client
	prefix string
}

//nolint:ireturn // required by broker.Register.
func newRedis(ctx context.Context, connst string) (broker.Broker, error) {
	u, err := url.Parse(connst)
	if err != nil {
		return nil, errorx.Wrap(err, "parse connection url")
	}

	cl, err := gredis.New(ctx, connst)
	if err != nil {
		return nil, errorx.Wrap(err, "create redis client")
	}

	return &Broker{
		cl:     cl,
		prefix: u.Fragment,
	}, nil
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

func (b *Broker) Publish(ctx context.Context, channel string, payload any) error {
	channel = b.prefix + ":" + channel

	bs, err := json.Marshal(payload)
	if err != nil {
		return errorx.Wrap(err, "marshal payload")
	}

	zerolog.Ctx(ctx).Debug().Str("payload", string(bs)).Str("channel", channel).
		Msg("Publishing message")

	err = b.cl.Publish(ctx, channel, bs).Err()

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

loop:
	for {
		var msg *redis.Message

		select {
		case <-ctx.Done():
			break loop
		case rmsg, ok := <-ch:
			if !ok {
				break loop
			}

			msg = rmsg
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

			err := mh(ctx, msg.Channel, []byte(msg.Payload))
			if err != nil {
				logger.Error().Err(err).Msg("Error processing message")
			}
		}()
	}

	wg.Wait()
}
