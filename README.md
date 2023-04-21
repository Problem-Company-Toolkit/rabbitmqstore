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