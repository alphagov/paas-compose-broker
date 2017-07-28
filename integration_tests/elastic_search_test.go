package integration_test

import (
	. "github.com/onsi/ginkgo"

	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("Broker Compose Integration", func() {

	Context("ElasticSearch", func() {

		BeforeEach(func() {
			if skipIntegrationTests {
				Skip("SKIP_COMPOSE_API_TESTS is set, skipping tests against real Compose API")
			}
		})

		It("should support the full instance lifecycle", func() {

			const (
				elasticSearchServiceID = "6e9202f2-c2e1-4de8-8d4a-a8c898fc2d8c"
				elasticSearchPlanID    = "6d051078-0913-403c-9763-1d03ecee50d9"
			)

			var (
				service *helper.ServiceHelper
				binding *helper.BindingData
			)

			By("initializing service from catalog", func() {
				service = helper.NewService(elasticSearchServiceID, elasticSearchPlanID)
			})

			By("provisioning the service", func() {
				service.Provision()
			})

			defer By("deprovisoning the service", func() {
				service.Deprovision()
			})

			By("binding a resource to service", func() {
				binding = service.Bind()
			})

			defer By("unbinding the service", func() {
				service.Unbind(binding.ID)
			})

			// TODO: By("connecting to the service", func() { })

			// TODO: defer By("disconnecting from the service", func() { })

			// TODO: By("ensuring credentials allow writing data", func() { })

			// TODO: By("ensuring credentials allow reading data", func() { })

		})

	})

})
