package broker

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker", func() {
	Describe("squashErrors", func() {
		It("can squash errors", func() {
			errors := []error{
				errors.New("first"),
				errors.New("second"),
			}
			Expect(squashErrors(errors)).To(MatchError("first; second"))
		})
	})

	Describe("JDBCURI", func() {
		It("can create JDBC URI", func() {
			uri := JDBCURI("mongo", "host-a.com", "1111", "admin", "username", "password")
			Expect(uri).To(Equal("jdbc:mongodb://host-a.com:1111/admin?user=username&password=password"))
		})
	})

	Describe("makeOperationData", func() {
		It("can make operation data JSON", func() {
			operationData, err := makeOperationData("expected_type", "123")
			Expect(err).ToNot(HaveOccurred())
			Expect(operationData).To(Equal(`{"recipe_id":"123","type":"expected_type"}`))
		})
	})
})
