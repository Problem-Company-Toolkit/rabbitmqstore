package rabbitmqstore_test

import (
	"context"
	"os"

	"github.com/brianvoe/gofakeit/v6"
	rabbithole "github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rabbitmq/amqp091-go"

	"github.com/problem-company-toolkit/rabbitmqstore"
)

var _ = Describe("Rabbitmqstore", func() {
	var (
		store   rabbitmqstore.Store
		adminMq *rabbithole.Client
		url     string
	)

	BeforeEach(func() {
		var err error
		url = os.Getenv("RABBITMQ_ADDRESS")

		user := os.Getenv("RABBITMQ_DEFAULT_USER")
		password := os.Getenv("RABBITMQ_DEFAULT_PASS")

		adminMq, err = rabbithole.NewClient(os.Getenv("RABBITMQ_HTTP_ADDRESS"), user, password)
		Expect(err).ShouldNot(HaveOccurred())

		options := rabbitmqstore.Options{
			URL: url,
		}
		store, err = rabbitmqstore.New(options)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		store.CloseAll()
	})

	Context("Registering different listeners for each exchange", func() {
		It("should invoke the handler for the correct exchange", func() {
			exchangeName1 := gofakeit.Word()
			queueName1 := gofakeit.Word()
			bindingKey1 := gofakeit.Word()
			received1 := make(chan string, 1)

			exchangeName2 := gofakeit.Word()
			queueName2 := gofakeit.Word()
			bindingKey2 := gofakeit.Word()
			received2 := make(chan string, 1)

			handler1 := func(d amqp091.Delivery) {
				received1 <- exchangeName1
			}

			handler2 := func(d amqp091.Delivery) {
				received2 <- exchangeName2
			}

			_, err := store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
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
				Exchange:   gofakeit.Word(),
				Queue:      gofakeit.Word(),
				RoutingKey: gofakeit.Word(),
				Handler:    func(d amqp091.Delivery) {},
			}

			opts2 = rabbitmqstore.RegisterListenerOpts{
				Exchange:   gofakeit.Word(),
				Queue:      gofakeit.Word(),
				RoutingKey: gofakeit.Word(),
				Handler:    func(d amqp091.Delivery) {},
			}

			var err error
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

			err := channel.PublishWithContext(context.TODO(), opts1.Exchange, opts1.RoutingKey, false, false, amqp091.Publishing{ContentType: "text/plain", Body: []byte("x")})
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

	Context("Connection errors", func() {
		Context("Channel error", func() {
			var (
				listenChan chan struct{}
				opts       rabbitmqstore.RegisterListenerOpts
				channel    *amqp091.Channel
				errChan    chan *amqp091.Error
			)

			BeforeEach(func() {
				listenChan = make(chan struct{}, 1)
				opts = rabbitmqstore.RegisterListenerOpts{
					Exchange:     gofakeit.Word(),
					Queue:        gofakeit.UUID(),
					RoutingKey:   gofakeit.Word(),
					ExchangeType: amqp091.ExchangeTopic,
					Handler: func(d amqp091.Delivery) {
						defer func() {
							listenChan <- struct{}{}
						}()

						d.Ack(true)
						d.Ack(true)
					},
				}

				channel = store.GetChannel()

				_, err := store.RegisterListener(opts)

				if err != nil {
					Fail(err.Error())
					return
				}

				errChan = channel.NotifyClose(make(chan *amqp091.Error))
			})

			It("should recreate channel correctly", func() {
				for i := 0; i < 2; i++ {
					err := channel.PublishWithContext(context.TODO(), opts.Exchange, opts.RoutingKey, false, false, amqp091.Publishing{
						ContentType: "text/plain",
						Body:        []byte("hello"),
					})

					if err != nil {
						Fail(err.Error())
						return
					}
				}

				Eventually(errChan).Should(Receive(Not(BeNil())))
				Eventually(listenChan).MustPassRepeatedly(2).Should(Receive())
			})

			It("should to publish messages without any issue", func() {
				totalMessages := gofakeit.IntRange(5, 20)

				for i := 0; i < totalMessages; i++ {
					err := store.Publish(rabbitmqstore.PublishOpts{
						Context:    context.TODO(),
						Exchange:   opts.Exchange,
						RoutingKey: opts.RoutingKey,
						Mandatory:  false,
						Immediate:  false,
						Message: amqp091.Publishing{
							ContentType: "text/plain",
							Body:        []byte("hello"),
						},
					})

					Expect(err).ShouldNot(HaveOccurred())
				}

				Eventually(errChan).Should(Receive(Not(BeNil())))
				Eventually(listenChan).MustPassRepeatedly(totalMessages).Should(Receive())
			})
		})

		Context("Base connection error", func() {
			It("should to reconnect automatically", func() {
				queue := gofakeit.UUID()
				exchange := gofakeit.Word()
				routingKey := gofakeit.Word()

				listenChan := make(chan struct{}, 1)
				_, err := store.RegisterListener(rabbitmqstore.RegisterListenerOpts{
					Exchange:     exchange,
					Queue:        queue,
					RoutingKey:   routingKey,
					ExchangeType: amqp091.ExchangeTopic,
					Handler: func(d amqp091.Delivery) {
						listenChan <- struct{}{}
					},
				})

				if err != nil {
					Fail(err.Error())
					return
				}

				connErrChan := make(chan *amqp091.Error)

				store.GetConnection().NotifyClose(connErrChan)

				connectionInfoChan := make(chan []rabbithole.ConnectionInfo)
				go func() {
					var (
						connectionInfo []rabbithole.ConnectionInfo
						err            error
					)

					for len(connectionInfo) <= 0 {
						connectionInfo, err = adminMq.ListConnections()
						if err != nil {
							Fail(err.Error())
							return
						}
					}

					connectionInfoChan <- connectionInfo
				}()

				connections := <-connectionInfoChan

				for _, connection := range connections {
					_, err := adminMq.CloseConnection(connection.Name)

					if err != nil {
						Fail(err.Error())
						return
					}
				}

				errChan := make(chan error, 1)

				go func() {
					for {
						err := store.GetChannel().PublishWithContext(context.TODO(), exchange, routingKey, false, false, amqp091.Publishing{
							ContentType: "text/plain",
							Body:        []byte("hello"),
						})

						errChan <- err

						if err == nil {
							break
						}
					}
				}()

				Eventually(errChan).Should(Receive(BeNil()))
				Eventually(connErrChan).Should(Receive())
				Eventually(listenChan).Should(Receive())
			})
		})
	})

	It("Declares exchanges", func() {
		err := store.DeclareExchanges([]rabbitmqstore.DeclareExchangeOpts{
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
