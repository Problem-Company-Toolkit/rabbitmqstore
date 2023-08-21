package rabbitmqstore

import (
	"fmt"
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type ListenerHandlerFunc = func(amqp091.Delivery)

type listener struct {
	mutex sync.Mutex

	id               string
	exchange         string
	exchangeType     string
	queue            string
	routingKey       string
	recreateListener chan struct{}
	handler          func(amqp091.Delivery)
}

type RegisterListenerOpts struct {
	Exchange     string
	Queue        string
	RoutingKey   string
	ExchangeType string
	Handler      func(amqp091.Delivery)
}

type Listener interface {
	GetID() string
	GetExchange() string
	GetQueueName() string
	GetBindingKey() string
	GetExchangeType() string
	UpdateHandler(ListenerHandlerFunc)
}

func configureChannel(channel *amqp091.Channel, opts RegisterListenerOpts) (<-chan amqp091.Delivery, error) {
	err := channel.ExchangeDeclare(opts.Exchange, opts.ExchangeType, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	q, err := channel.QueueDeclare(opts.Queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = channel.QueueBind(q.Name, opts.RoutingKey, opts.Exchange, false, nil)
	if err != nil {
		return nil, err
	}

	return channel.Consume(q.Name, "", false, false, false, false, nil)
}

func (r *rabbitmqStore) RegisterListener(opts RegisterListenerOpts) (Listener, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := fmt.Sprintf(opts.Exchange)
	if opts.Queue != "" {
		id = fmt.Sprintf("%s/%s", id, opts.Queue)
	}
	if opts.RoutingKey != "" {
		id = fmt.Sprintf("%s/%s", id, opts.RoutingKey)
	}

	if _, exists := r.listeners[id]; exists {
		return nil, fmt.Errorf("listener with id %s already exists", id)
	}

	if opts.ExchangeType == "" {
		opts.ExchangeType = "topic"
	}

	msgs, err := configureChannel(r.channel, opts)
	if err != nil {
		return nil, err
	}

	recreateListener := make(chan struct{})

	r.listeners[id] = &listener{
		mutex:            sync.Mutex{},
		id:               id,
		exchange:         opts.Exchange,
		exchangeType:     opts.ExchangeType,
		routingKey:       opts.RoutingKey,
		queue:            opts.Queue,
		recreateListener: recreateListener,
		handler:          opts.Handler,
	}

	go func() {
		// Abstracted into its own function so that we can get away with using defer
		// Returns true in order to break the listen.
		handleFunc := func(d amqp091.Delivery) bool {
			if _, ok := r.listeners[id]; !ok {
				return true
			}

			r.listeners[id].mutex.Lock()
			defer r.listeners[id].mutex.Unlock()

			handler := r.listeners[id].handler
			if handler != nil {
				handler(d)
			}

			d.Ack(false)

			return false
		}

		var logger = r.logger.With(zap.String("Listener ID", id))
		logger.Debug(
			"Initializing listener",
		)

		consumeMessages := func(msgs <-chan amqp091.Delivery) {
			for d := range msgs {
				logger.Debug("Received message", zap.String("Message", string(d.Body)))
				handleFunc(d)
			}
		}

		go func() {
			for {
				_, ok := <-recreateListener

				msgs, continueLoop := func() (<-chan amqp091.Delivery, bool) {
					r.mutex.Lock()
					defer r.mutex.Unlock()

					channelClosed := r.channel.IsClosed()
					connClosed := r.conn.IsClosed()

					// Exits if the developer closes manually
					if channelClosed || connClosed || !ok {
						logger.Debug("Stopped listener")
						return nil, false
					}

					msgs, err := configureChannel(r.channel, opts)

					if err != nil {
						logger.Debug(
							"Failed to reconfigure the channel",
							zap.Error(err),
							zap.Bool("channel closed", channelClosed),
							zap.Bool("connection closed", connClosed),
						)
						return nil, false
					}

					return msgs, true
				}()

				if !continueLoop {
					break
				}

				consumeMessages(msgs)
			}
		}()

		consumeMessages(msgs)
	}()

	return r.listeners[id], nil
}

func (l *listener) GetExchange() string {
	return l.exchange
}
func (l *listener) GetExchangeType() string {
	return l.exchangeType
}
func (l *listener) GetQueueName() string {
	return l.queue
}
func (l *listener) GetBindingKey() string {
	return l.routingKey
}
func (l *listener) GetID() string {
	return l.id
}
func (l *listener) UpdateHandler(fn ListenerHandlerFunc) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.handler = fn
}
