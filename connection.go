package rabbitmqstore

import (
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *rabbitmqStore) handleChannel() {
	for {
		reason, ok := <-r.channel.NotifyClose(make(chan *amqp091.Error))

		// Exits if the developer closes manually
		if !ok && r.channel.IsClosed() {
			r.logger.Debug("Channel closed")
			break
		}

		r.logger.Debug("Unexpected channel closed", zap.Error(reason))

		for {
			channel, err := r.conn.Channel()

			if err == nil {
				r.logger.Debug("Channel recreated successfully")

				r.mutex.Lock()
				r.channel = channel
				r.mutex.Unlock()
				break
			}

			r.logger.Debug("Failed to recreate the channel", zap.Error(err))
		}
	}
}
