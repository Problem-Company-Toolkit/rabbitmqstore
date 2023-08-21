package rabbitmqstore

import (
	"context"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type PublishOpts struct {
	Context    context.Context
	Exchange   string
	RoutingKey string

	Mandatory bool
	Immediate bool
	Message   amqp091.Publishing
}

func (r *rabbitmqStore) Publish(opts PublishOpts) error {
	r.logger.Debug(
		"Publishing message",
		zap.String("Exchange", opts.Exchange),
		zap.String("Routing Key", opts.RoutingKey),
		zap.String("Body", string(opts.Message.Body)),
	)

	// Waits for finish state changes in store to send the message.
	// It prevents trying to send a message while reconnecting.
	for !r.mutex.TryLock() {
	}
	r.mutex.Unlock()

	return r.channel.PublishWithContext(
		opts.Context,
		opts.Exchange,
		opts.RoutingKey,
		opts.Mandatory,
		opts.Immediate,
		opts.Message,
	)
}
