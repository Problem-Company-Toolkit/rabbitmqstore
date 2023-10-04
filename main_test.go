package rabbitmqstore_test

import (
	"context"
	"fmt"
	"os"

	"github.com/brianvoe/gofakeit/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rabbitmq/amqp091-go"

	"github.com/problem-company-toolkit/rabbitmqstore"
)

func newFriendlyName() string {
	return fmt.Sprintf("%s-%d", gofakeit.Word(), gofakeit.IntRange(1000000, 9999999))
}

var _ = Describe("Rabbitmqstore", func() {
	var (
		store      rabbitmqstore.Store
		err        error
		connection *amqp091.Connection
		url        string
	)

	BeforeEach(func() {
		url = fmt.Sprintf(
			"amqp://%s:%s@%s:%s/",
			os.Getenv("RABBITMQ_DEFAULT_USER"),
			os.Getenv("RABBITMQ_DEFAULT_PASS"),
			os.Getenv("RABBITMQ_HOST"),
			os.Getenv("RABBITMQ_PORT"),
		)
		options := rabbitmqstore.Options{
			URL: url,
		}
		store, err = rabbitmqstore.New(options)
		Expect(err).NotTo(HaveOccurred())

		connection, err = amqp091.Dial(options.URL)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		store.CloseAll()
		connection.Close()
	})

	Context("single exchange, many listeners", func() {
		var (
			exchange           string
			queue              string
			routingKeysList    []string
			listenersMap       map[string]func(d amqp091.Delivery)
			routingKeyChannels map[string]chan string

			channel *amqp091.Channel
		)

		const ROUTING_KEYS_COUNT = 10

		BeforeEach(func() {

			exchange = newFriendlyName()
			queue = newFriendlyName()

			err := store.DeclareExchanges([]rabbitmqstore.DeclareExchangeOpts{
				{
					Exchange: exchange,
					Kind:     "topic",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			channel = store.GetChannel()
			Expect(channel).ToNot(BeNil())

			routingKeysList = []string{}
			routingKeyChannels = map[string]chan string{}
			listenersMap = map[string]func(d amqp091.Delivery){}

			for i := 0; i < ROUTING_KEYS_COUNT; i++ {
				routingKey := newFriendlyName()

				routingKeysList = append(routingKeysList, routingKey)

				routingKeyChannels[routingKey] = make(chan string, 1)

				handler := func(d amqp091.Delivery) {
					routingKeyChannels[routingKey] <- string(d.Body)
				}
				listenersMap[routingKey] = handler
				_, err = store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
					Exchange:   exchange,
					Queue:      queue,
					RoutingKey: routingKey,
					Handler:    listenersMap[routingKey],
				})
				Expect(err).ShouldNot(HaveOccurred())
			}
		})

		It("triggers the correct listener when there are many", func() {
			// Never the first, never the last.
			randomIndex := gofakeit.IntRange(1, len(routingKeysList)-2)
			routingKey := routingKeysList[randomIndex]

			// Do it a bunch of times to be sure it can reliably do so.
			const REPEAT_TIMES = 5

			for i := 0; i < REPEAT_TIMES; i++ {
				expectedValue := fmt.Sprintf("iteration: %d", i)

				err = channel.PublishWithContext(context.TODO(), exchange, routingKey, false, false, amqp091.Publishing{ContentType: "text/plain", Body: []byte(expectedValue)})
				Expect(err).NotTo(HaveOccurred())

				Eventually(routingKeyChannels[routingKey]).Should(Receive(&expectedValue))
			}
		})
	})

	Context("Registering different listeners for each exchange", func() {
		var (
			exchangeName1 string
			queueName1    string
			bindingKey1   string

			exchangeName2 string
			queueName2    string
			bindingKey2   string
		)

		BeforeEach(func() {
			exchangeName1 = newFriendlyName()
			queueName1 = newFriendlyName()
			bindingKey1 = newFriendlyName()

			exchangeName2 = newFriendlyName()
			queueName2 = newFriendlyName()
			bindingKey2 = newFriendlyName()

		})

		It("should invoke the handler for the correct exchange", func() {
			received1 := make(chan string, 1)
			received2 := make(chan string, 1)
			handler1 := func(d amqp091.Delivery) {
				received1 <- exchangeName1
			}

			handler2 := func(d amqp091.Delivery) {
				received2 <- exchangeName2
			}

			_, err = store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
				Exchange:   exchangeName1,
				Queue:      queueName1,
				RoutingKey: bindingKey1,
				Handler:    handler1,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
				Exchange:   exchangeName2,
				Queue:      queueName2,
				RoutingKey: bindingKey2,
				Handler:    handler2,
			})
			Expect(err).NotTo(HaveOccurred())

			channel := store.GetChannel()
			Expect(err).NotTo(HaveOccurred())

			err = channel.PublishWithContext(context.TODO(), exchangeName1, bindingKey1, false, false, amqp091.Publishing{ContentType: "text/plain", Body: []byte("x")})
			Expect(err).NotTo(HaveOccurred())

			err = channel.PublishWithContext(context.TODO(), exchangeName2, bindingKey2, false, false, amqp091.Publishing{ContentType: "text/plain", Body: []byte("y")})
			Expect(err).NotTo(HaveOccurred())

			Eventually(received1).Should(Receive(&exchangeName1))
			Eventually(received2).Should(Receive(&exchangeName2))
		})
	})

	Context("Listeners management", func() {
		var (
			listener1, listener2 rabbitmqstore.Listener
			opts1, opts2         rabbitmqstore.RegisterListenerOpts
		)

		BeforeEach(func() {
			opts1 = rabbitmqstore.RegisterListenerOpts{
				Exchange:   newFriendlyName(),
				Queue:      newFriendlyName(),
				RoutingKey: newFriendlyName(),
				Handler:    func(d amqp091.Delivery) {},
			}

			opts2 = rabbitmqstore.RegisterListenerOpts{
				Exchange:   newFriendlyName(),
				Queue:      newFriendlyName(),
				RoutingKey: newFriendlyName(),
				Handler:    func(d amqp091.Delivery) {},
			}

			listener1, err = store.RegisterListener(opts1)
			Expect(err).NotTo(HaveOccurred())

			listener2, err = store.RegisterListener(opts2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should add listeners to the map", func() {
			listeners := store.GetListeners()

			Expect(len(listeners)).To(Equal(2))
			Expect(listeners[listener1.GetID()]).To(Equal(listener1))
			Expect(listeners[listener2.GetID()]).To(Equal(listener2))
		})

		It("should update the handler function", func() {
			received := make(chan string, 1)
			expectedMessage := gofakeit.UUID()
			newHandler := func(d amqp091.Delivery) {
				received <- expectedMessage
			}

			listener1.UpdateHandler(newHandler)

			channel := store.GetChannel()
			Expect(err).NotTo(HaveOccurred())

			err = channel.PublishWithContext(context.TODO(), opts1.Exchange, opts1.RoutingKey, false, false, amqp091.Publishing{ContentType: "text/plain", Body: []byte("x")})
			Expect(err).NotTo(HaveOccurred())

			Eventually(received).Should(Receive(&expectedMessage))
		})

		It("should deregister listeners", func() {
			store.CloseListener(listener1.GetID())

			listeners := store.GetListeners()
			Expect(len(listeners)).To(Equal(1))
			Expect(listeners[listener2.GetID()]).To(Equal(listener2))
		})
	})

	It("Declares exchanges", func() {
		err = store.DeclareExchanges([]rabbitmqstore.DeclareExchangeOpts{
			{
				Exchange: gofakeit.UUID(),
			},
			{
				Exchange: gofakeit.UUID(),
			},
			{
				Exchange: gofakeit.UUID(),
			},
			{
				Exchange: gofakeit.UUID(),
			},
		})
		Expect(err).ShouldNot(HaveOccurred())
	})
})
