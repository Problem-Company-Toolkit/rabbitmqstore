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

	id           string
	exchange     string
	exchangeType string
	queue        string
	bindingKey   string
	handler      func(amqp091.Delivery)
}

type RegisterListenerOpts struct {
	Exchange     string
	Queue        string
	BindingKey   string
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

func (r *rabbitmqStore) RegisterListener(opts RegisterListenerOpts) (Listener, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id := fmt.Sprintf(opts.Exchange)
	if opts.Queue != "" {
		id = fmt.Sprintf("%s-%s", id, opts.Queue)
	}
	if opts.BindingKey != "" {
		id = fmt.Sprintf("%s-%s", id, opts.BindingKey)
	}

	if _, exists := r.listeners[id]; exists {
		return nil, fmt.Errorf("listener with id %s already exists", id)
	}

	if opts.ExchangeType == "" {
		opts.ExchangeType = "topic"
	}

	err := r.channel.ExchangeDeclare(opts.Exchange, opts.ExchangeType, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	q, err := r.channel.QueueDeclare(opts.Queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = r.channel.QueueBind(q.Name, opts.BindingKey, opts.Exchange, false, nil)
	if err != nil {
		return nil, err
	}

	msgs, err := r.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	r.listeners[id] = &listener{
		mutex:        sync.Mutex{},
		id:           id,
		exchange:     opts.Exchange,
		exchangeType: opts.ExchangeType,
		bindingKey:   opts.BindingKey,
		queue:        opts.Queue,
		handler:      opts.Handler,
	}

	go func() {
		// Abstracted into its own function so that we can get away with using defer
		// Returns true in order to break the listen.
		handleFunc := func(d amqp091.Delivery) bool {
			if _, ok := r.listeners[id]; !ok {
				return true
			}

			r.listeners[id].mutex.Lock()

			r.listeners[id].handler(d)
			d.Ack(false)

			r.listeners[id].mutex.Unlock()
			return false
		}

		r.logger.Debug(
			"Initializing listener",
			zap.String("ID", id),
		)

		for d := range msgs {
			handleFunc(d)
		}
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
	return l.bindingKey
}
func (l *listener) GetID() string {
	return l.id
}
func (l *listener) UpdateHandler(fn ListenerHandlerFunc) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.handler = fn
}
