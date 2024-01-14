.PHONY: dependencies test-dependencies test

dependencies:
	@go mod download

test-dependencies:
	@go install github.com/onsi/ginkgo/v2/ginkgo@v2.13.2 \
	&& go install github.com/golang/mock/mockgen@v1.6.0

test:
	@ginkgo -r