package broker

import (
	"context"
	"encoding/json"

	"github.com/sovamorco/errorx"
)

type MessageHandler func(ctx context.Context, channel string, payload []byte) error

type Broker interface {
	Subscribe(ctx context.Context, mh MessageHandler, channels ...string)
	Publish(ctx context.Context, channel string, payload any) error
	Shutdown(ctx context.Context)
}

type StructHandler[T any] func(ctx context.Context, channel string, payload T) error

func StructToMessageHandler[T any](sh StructHandler[T]) MessageHandler {
	return func(ctx context.Context, channel string, payload []byte) error {
		var s T

		err := json.Unmarshal(payload, &s)
		if err != nil {
			return errorx.Wrap(err, "unmarshal payload")
		}

		return sh(ctx, channel, s)
	}
}
