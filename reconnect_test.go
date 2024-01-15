package rabbitmqstore_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rabbitmq/amqp091-go"

	"github.com/problem-company-toolkit/rabbitmqstore"
)

var _ = Describe("Reconnect", func() {
	var (
		store rabbitmqstore.Store

		exchangeName string
	)

	const (
		ACCEPTABLE_DELAY = time.Second * 10
	)

	BeforeEach(func() {
		var err error
		options := rabbitmqstore.Options{
			URL: getUrl(),
		}
		store, err = rabbitmqstore.New(options)
		Expect(err).NotTo(HaveOccurred())

		exchangeName = newFriendlyName()
		err = store.DeclareExchanges([]rabbitmqstore.DeclareExchangeOpts{
			{
				Exchange: exchangeName,
				Durable:  false,
				Kind:     "topic",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
			Exchange: exchangeName,
			Handler: func(d amqp091.Delivery) {
				d.Ack(false)
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Reconnect", func() {
		It("should reconnect", func() {
			err := store.CloseAll()
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second * 1)

			Expect(store.GetChannel().IsClosed()).To(BeTrue())

			// Make sure the reconnect logic isn't triggering a new connection after an explicit close.
			Eventually(store.GetChannel().IsClosed(), ACCEPTABLE_DELAY).ShouldNot(BeFalse())

			err = store.Reconnect()
			Expect(err).NotTo(HaveOccurred())
			Eventually(store.GetChannel().IsClosed(), ACCEPTABLE_DELAY).Should(BeFalse())
		})
	})

	Context("An exception occurs", func() {
		It("should reconnect", func() {
			messageSent := make(chan interface{})
			_, err := store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
				Exchange:   exchangeName,
				RoutingKey: "test-with-exception-occurs",
				Handler: func(d amqp091.Delivery) {
					// Double acknowledge triggers an exception that closes the channel
					d.Ack(false)
					d.Ack(false)
					messageSent <- true
				},
			})
			Expect(err).ToNot(HaveOccurred())

			err = store.Publish(rabbitmqstore.PublishOpts{
				Exchange:   exchangeName,
				RoutingKey: "test-with-exception-occurs",
				Context:    context.TODO(),
				Message: amqp091.Publishing{
					Body: []byte("test-with-exception-occurs"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			// Crash occurred
			Eventually(messageSent).Should(Receive())

			// Channel was restored
			Eventually(func(g Gomega) {
				isOpen := !store.GetChannel().IsClosed()
				g.Expect(isOpen).To(BeTrue())
			}, ACCEPTABLE_DELAY).Should(Succeed())
		})
	})
})
