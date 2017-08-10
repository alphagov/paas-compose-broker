package dbengine

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider Service", func() {
	var (
		provider *ProviderService
	)

	BeforeEach(func() {
		provider = NewProviderService()
	})

	Describe("GetSQLEngine", func() {
		It("returns error if engine is not supported", func() {
			_, err := provider.GetDBEngine("unknown")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("DB Engine 'unknown' not supported"))
		})

		Context("when engine is mongodb", func() {
			It("return the proper SQL Engine", func() {
				engine, err := provider.GetDBEngine("mongodb")
				Expect(err).ToNot(HaveOccurred())
				Expect(engine).To(BeAssignableToTypeOf(&MongoEngine{}))
			})
		})
	})
})
