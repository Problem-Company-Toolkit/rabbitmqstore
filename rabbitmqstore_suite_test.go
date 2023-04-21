package rabbitmqstore_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRabbitmqstore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rabbitmqstore Suite")
}
