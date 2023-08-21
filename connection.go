package rabbitmqstore

import (
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func (r *rabbitmqStore) notifyListeners() {
	for _, mqListener := range r.listeners {
		go func(mqListener *listener) {
			mqListener.recreateListener <- struct{}{}
		}(mqListener)
	}
}

func (r *rabbitmqStore) closeListeners() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for id := range r.listeners {
		r.CloseListener(id)
	}
}

func (r *rabbitmqStore) handleChannel() {
	for {
		reason, ok := <-r.channel.NotifyClose(make(chan *amqp091.Error))

		// Exits if the developer closes manually
		if !ok && r.channel.IsClosed() {
			r.logger.Debug("Channel closed")
			r.closeListeners()
			break
		}

		r.logger.Debug("Unexpected channel closed", zap.Error(reason))

		for {
			channel, err := r.conn.Channel()

			if err == nil {
				r.mutex.Lock()
				r.channel = channel
				r.mutex.Unlock()
				r.notifyListeners()
				r.logger.Debug("Channel recreated successfully")
				break
			}

			r.logger.Debug("Failed to recreate the channel", zap.Error(err))
		}
	}
}

func (r *rabbitmqStore) handleConnection() {
	for {
		reason, ok := <-r.conn.NotifyClose(make(chan *amqp091.Error))

		// Exits if closed manually
		if !ok {
			r.logger.Debug("Connection closed")
			r.closeListeners()
			break
		}

		r.logger.Debug("Unexpected connection closed", zap.Error(reason))

		for {
			conn, err := amqp091.Dial(r.url)

			if err != nil {
				r.logger.Debug("Failed to reconnect", zap.Error(err))
				continue
			}

			r.logger.Debug("Reconnected successfully")

			channel, err := conn.Channel()

			if err == nil {
				r.logger.Debug("Recreated channel successfully")

				r.mutex.Lock()
				r.conn = conn
				r.channel = channel
				r.mutex.Unlock()
				r.notifyListeners()
				break
			}

			r.logger.Debug("Failed to recreate lost channel")
		}
	}
}
