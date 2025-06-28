package broker

import (
	"context"
)

type MessageHandler func(ctx context.Context, channel, payload string) error

type Broker interface {
	Subscribe(ctx context.Context, mh MessageHandler, channels ...string)
	Publish(ctx context.Context, channel, payload string) error
	Shutdown(ctx context.Context)
}
