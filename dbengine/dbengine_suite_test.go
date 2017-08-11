package dbengine_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDbengine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DBEngine Suite")
}
