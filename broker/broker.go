package broker

import (
	"context"
)

type MessageHandler func(ctx context.Context, channel string, payload []byte) error

type Broker interface {
	Subscribe(ctx context.Context, mh MessageHandler, channels ...string)
	Publish(ctx context.Context, channel string, payload any) error
	Shutdown(ctx context.Context)
}
