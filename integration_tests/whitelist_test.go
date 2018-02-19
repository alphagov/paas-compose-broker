package integration_test

import (
	"time"

	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/client/elastic"
	"github.com/alphagov/paas-compose-broker/dbengine"
	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("whitelisting deployments", func() {

	const (
		elasticSearchServiceID = "6e9202f2-c2e1-4de8-8d4a-a8c898fc2d8c"
		elasticSearchPlanID    = "6d051078-0913-403c-9763-1d03ecee50d9"
	)

	var (
		service    *helper.ServiceHelper
		instanceID string
		binding    *helper.BindingData
		appID      string
		client     *elastic.Client
	)

	BeforeEach(func() {
		if skipIntegrationTests {
			Skip("SKIP_COMPOSE_API_TESTS is set, skipping tests against real Compose API")
		}

		appID = helper.NewUUID()
	})

	It("should support whitelisting IPs", func() {
		By("initializing service from catalog", func() {
			service = helper.NewService(elasticSearchServiceID, elasticSearchPlanID, []string{"1.1.1.1"})
		})

		By("provisioning a service", func() {
			instanceID = service.Provision(map[string]interface{}{})
		})

		defer By("deprovisioning the service", func() {
			service.Deprovision(instanceID)
		})

		By("binding a resource to the service", func() {
			binding = service.Bind(instanceID, appID)
		})

		defer By("unbinding the service", func() {
			service.Unbind(instanceID, binding.ID)
		})

		By("ensuring that whitelist is set", func() {
			deploymentName, err := broker.MakeInstanceName(service.Cfg.DBPrefix, instanceID)
			Expect(err).NotTo(HaveOccurred())

			var whitelist []composeapi.DeploymentWhitelist
			Eventually(func() []error {
				var errs []error
				deployment, errs := service.ComposeClient.GetDeploymentByName(deploymentName)
				if errs != nil {
					return errs
				}
				whitelist, errs = service.ComposeClient.GetWhitelistForDeployment(deployment.ID)
				return errs
			}, 1*time.Minute, 15*time.Second).Should(BeEmpty())

			Expect(len(whitelist)).To(Equal(1))
			Expect(whitelist[0].IP).To(Equal("1.1.1.1/32"))
		})

		By("ensuring that access is denied", func() {
			httpClient, err := dbengine.SetupHTTPClient(binding.Credentials.CACertificateBase64)
			Expect(err).NotTo(HaveOccurred())

			client, err = elastic.New(binding.Credentials.URI, httpClient)
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Version()
			Expect(err).To(HaveOccurred())
		})
	})
})
