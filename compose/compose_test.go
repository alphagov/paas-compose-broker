package compose

import (
	"errors"

	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var clusters []composeapi.Cluster

var _ = Describe("Compose", func() {
	Describe("squashErrors", func() {
		It("can squash errors", func() {
			errors := []error{
				errors.New("first"),
				errors.New("second"),
			}
			Expect(SquashErrors(errors)).To(MatchError("first; second"))
		})
	})
})
