package compose_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCompose(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compose Suite")
}
