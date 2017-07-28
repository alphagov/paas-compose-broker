package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"testing"
)

var skipIntegrationTests = os.Getenv("SKIP_COMPOSE_API_TESTS") == "true"

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Integration Tests")
}
