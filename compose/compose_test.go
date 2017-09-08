package compose

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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
