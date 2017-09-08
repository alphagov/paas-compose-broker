package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
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
			engine, err := provider.GetDBEngine(&composeapi.Deployment{Type: "unknown"})
			Expect(err).To(HaveOccurred())
			Expect(engine).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("DB Engine 'unknown' not supported"))
		})

		Context("when engine is mongodb", func() {
			It("return the proper SQL Engine", func() {
				engine, err := provider.GetDBEngine(&composeapi.Deployment{Type: "mongodb"})
				Expect(err).ToNot(HaveOccurred())
				Expect(engine).To(BeAssignableToTypeOf(&MongoEngine{}))
			})
		})
	})
})
