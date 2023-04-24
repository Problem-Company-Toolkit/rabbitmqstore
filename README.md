# RabbitMQ Store

`rabbitmqstore` is a Go package designed to simplify the usage of RabbitMQ for message-based communication between microservices. It provides an easy-to-use interface to create, manage, and interact with listeners for specific exchanges and queues, allowing developers to focus on implementing business logic rather than dealing with the intricacies of setting up and managing RabbitMQ connections and channels.

## Features

- Connection and channel management
- Listener registration for specific exchanges, queues, and binding keys
- Listener deregistration
- Handler function updates for registered listeners
- Access to the underlying RabbitMQ channel for custom operations

## Installation

To install the `rabbitmqstore` package, use the following command:

```
go get -u github.com/problem-company-toolkit/rabbitmqstore
```

## Usage

### Import the package

```go
import (
	"github.com/problem-company-toolkit/rabbitmqstore"
)
```

### Create a new RabbitMQ store

Create a new RabbitMQ store instance by providing the RabbitMQ connection URL:

```go
options := rabbitmqstore.Options{
	URL: "amqp://guest:guest@localhost:5672/",
}

store, err := rabbitmqstore.New(options)
if err != nil {
	panic(err)
}
```

### Register a listener

Register a listener to handle messages for a specific exchange, queue, and binding key:

```go
opts := rabbitmqstore.RegisterListenerOpts{
	Exchange:     "my.exchange",
	Queue:        "my.queue",
	BindingKey:   "my.binding.key",
	ExchangeType: "topic", // optional, defaults to "topic"
	Handler: func(d amqp091.Delivery) {
		fmt.Printf("Received message: %s\n", string(d.Body))
	},
}

listener, err := store.RegisterListener(opts)
if err != nil {
	panic(err)
}
```

### Update the handler function

Update the handler function for an existing listener:

```go
newHandler := func(d amqp091.Delivery) {
	fmt.Printf("New handler received message: %s\n", string(d.Body))
}

listener.UpdateHandler(newHandler)
```

### Deregister a listener

Deregister a listener to stop processing messages for the associated exchange, queue, and binding key:

```go
store.CloseListener(listener.GetID())
```

### Access the underlying RabbitMQ channel

Access the underlying RabbitMQ channel to perform custom operations:

```go
channel := store.GetChannel()

// Perform custom operations using the channel
```

### Close all connections

Close all RabbitMQ connections and channels associated with the store:

```go
err = store.CloseAll()
if err != nil {
	panic(err)
}
```

## Logging

`rabbitmqstore` provides customizable logging capabilities to keep track of the events and errors occurring in the package. The logger can be configured using the `LoggerOpts` field in the `Options` struct, allowing you to set the desired log level, log encoding, and custom logger.

```go
import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Custom logger configuration
loggerConfig := zap.Config{
	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
	Development:       false,
	DisableCaller:     true,
	DisableStacktrace: true,
	OutputPaths:       []string{"stdout"},
	ErrorOutputPaths:  []string{"stderr"},
	Encoding:          "json",
	EncoderConfig: zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		CallerKey:      "caller",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	},
}
logger, err := loggerConfig.Build()
if err != nil {
	panic(err)
}

options := rabbitmqstore.Options{
	URL: "amqp://guest:guest@localhost:5672/",
	LoggerOpts: rabbitmqstore.LoggerOptions{
		Logger: logger,
	},
}

store, err := rabbitmqstore.New(options)
if err != nil {
	panic(err)
}
```

The package uses a default log level of `WarnLevel` and a default log encoding of `console`. You can also set the log level using the environment variable `RABBITMQSTORE_LOG_LEVEL`. Supported log levels are `debug`, `info`, `warn`, `fatal`, `panic`, and `dpanic`. If an invalid log level is provided, the default log level (`WarnLevel`) will be used.

If the log level is set to `debug`, then `rabbitmqstore` will log all publish and received messages, including their contents.