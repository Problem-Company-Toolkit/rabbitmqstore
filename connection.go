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

func (r *rabbitmqStore) handleReconnect() {
	for {
		var continueLoop bool
		select {
		case reason, ok := <-r.channel.NotifyClose(make(chan *amqp091.Error)):
			continueLoop = r.handleChannel(reason, ok)
		case reason, ok := <-r.conn.NotifyClose(make(chan *amqp091.Error)):
			continueLoop = r.handleConnection(reason, ok)
		}

		if !continueLoop {
			break
		}
	}
}

func (r *rabbitmqStore) handleChannel(reason *amqp091.Error, ok bool) bool {
	// Exits if the developer closes manually
	r.logger.Debug(
		"close status",
		zap.Bool("channel closed", r.channel.IsClosed()),
		zap.Bool("err chan open", ok),
	)
	if !ok && r.channel.IsClosed() {
		r.logger.Debug("Channel closed")
		r.closeListeners()
		return false
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

	return true
}

func (r *rabbitmqStore) handleConnection(reason *amqp091.Error, ok bool) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Exits if closed manually
	if !ok {
		r.logger.Debug("Connection closed")
		r.closeListeners()
		return false
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

			r.conn = conn
			r.channel = channel
			r.notifyListeners()
			break
		}

		r.logger.Debug("Failed to recreate lost channel")
	}

	return true
}
