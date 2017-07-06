package broker

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker utility functions", func() {

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

	Describe("makeInstanceName", func() {
		It("can make an instance name", func() {
			instanceName, err := makeInstanceName("test", "15e332e8-4afa-4c41-82a3-f44b18eba448")
			Expect(err).ToNot(HaveOccurred())
			Expect(instanceName).To(Equal("test-15e332e8-4afa-4c41-82a3-f44b18eba448"))
		})

		It("can trim spaces from dbprefix", func() {
			instanceName, err := makeInstanceName(" trim-spaces ", "0f38f9c2-085c-41ec-87bf-e38b72f7fdaa")
			Expect(err).ToNot(HaveOccurred())
			Expect(instanceName).To(Equal("trim-spaces-0f38f9c2-085c-41ec-87bf-e38b72f7fdaa"))
		})
	})
})
