package rabbitmqstore

import (
	"sync"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type MessageHandler func(amqp091.Delivery)

type Store interface {
	RegisterListener(RegisterListenerOpts) (Listener, error)
	CloseListener(id string)
	GetListeners() map[string]Listener

	// Declares a list of exchanges. Useful for initializing the exchanges that the store will use.
	DeclareExchanges([]DeclareExchangeOpts) error

	CloseAll() error
	GetChannel() *amqp091.Channel
}

type rabbitmqStore struct {
	mutex     sync.Mutex
	logger    *zap.Logger
	conn      *amqp091.Connection
	channel   *amqp091.Channel
	listeners map[string]*listener
}

func (r *rabbitmqStore) GetChannel() *amqp091.Channel {
	return r.channel
}

type Options struct {
	// Required if Connection is not provided.
	URL string

	// Required if URL is not provided.
	Connection *amqp091.Connection
}

func New(options Options) (Store, error) {
	var conn *amqp091.Connection = options.Connection
	var err error

	if conn == nil {
		conn, err = amqp091.Dial(options.URL)
		if err != nil {
			return nil, err
		}
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	return &rabbitmqStore{
		mutex:     sync.Mutex{},
		logger:    logger,
		conn:      conn,
		channel:   channel,
		listeners: make(map[string]*listener),
	}, nil
}

func (r *rabbitmqStore) GetListeners() map[string]Listener {
	listeners := make(map[string]Listener)
	for k := range r.listeners {
		listeners[k] = r.listeners[k]
	}
	return listeners
}

func (r *rabbitmqStore) CloseAll() error {
	err := r.channel.Close()
	if err != nil {
		return err
	}

	err = r.conn.Close()
	if err != nil {
		return err
	}

	return nil
}

// Safe to call for IDs that don't exist.
func (r *rabbitmqStore) CloseListener(id string) {
	_, ok := r.listeners[id]
	if !ok {
		return
	}

	r.listeners[id].mutex.Lock()
	delete(r.listeners, id)
}

type DeclareExchangeOpts struct {
	// Required.
	Exchange string

	// Defaults to topic.
	Kind string
}

func (r *rabbitmqStore) DeclareExchanges(optsList []DeclareExchangeOpts) error {
	for i := range optsList {
		opt := optsList[i]

		if opt.Kind == "" {
			opt.Kind = "topic"
		}

		r.logger.Debug(
			"Declaring exchange",
			zap.String("Exchange", opt.Exchange),
			zap.String("Kind", opt.Kind),
		)

		if err := r.channel.ExchangeDeclare(
			opt.Exchange,
			opt.Kind,
			true,
			false,
			false,
			false,
			nil,
		); err != nil {
			return err
		}
	}

	return nil
}
