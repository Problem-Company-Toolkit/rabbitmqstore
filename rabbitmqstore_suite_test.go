package rabbitmqstore_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func getUrl() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%s/",
		os.Getenv("RABBITMQ_DEFAULT_USER"),
		os.Getenv("RABBITMQ_DEFAULT_PASS"),
		os.Getenv("RABBITMQ_HOST"),
		os.Getenv("RABBITMQ_PORT"),
	)
}

func newFriendlyName() string {
	return fmt.Sprintf("%s-%d", gofakeit.Word(), gofakeit.IntRange(1000000, 9999999))
}

func TestRabbitmqstore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rabbitmqstore Suite")
}
