package rabbitmqstore_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rabbitmq/amqp091-go"

	"github.com/problem-company-toolkit/rabbitmqstore"
)

var _ = Describe("Reconnect", func() {
	var (
		store rabbitmqstore.Store
	)

	BeforeEach(func() {
		var err error
		options := rabbitmqstore.Options{
			URL: getUrl(),
		}
		store, err = rabbitmqstore.New(options)
		Expect(err).NotTo(HaveOccurred())

		exchange1 := newFriendlyName()
		err = store.DeclareExchanges([]rabbitmqstore.DeclareExchangeOpts{
			{
				Exchange: exchange1,
				Durable:  false,
				Kind:     "topic",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
			Exchange: exchange1,
			Handler: func(d amqp091.Delivery) {
				d.Ack(false)
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reconnect", func() {
		err := store.CloseAll()
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 1)

		Expect(store.GetChannel().IsClosed()).To(BeTrue())

		err = store.Reconnect()
		Expect(err).NotTo(HaveOccurred())
		Expect(store.GetChannel().IsClosed()).To(BeFalse())
	})
})
