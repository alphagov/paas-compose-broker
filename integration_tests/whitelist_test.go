package integration_test

import (
	"github.com/garyburd/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("whitelisting deployments", func() {

	const (
		redisServiceID = "1356eeeb-7c5d-4d9d-9a04-c035a2c709b3"
		redisPlanID    = "a8574a4b-9c6c-40ea-a0df-e9b7507948c8"
	)

	var (
		service *helper.ServiceHelper
		binding *helper.BindingData
		conn    redis.Conn
		appID   string
	)

	BeforeEach(func() {
		if skipIntegrationTests {
			Skip("SKIP_COMPOSE_API_TESTS is set, skipping tests against real Compose API")
		}

		appID = helper.NewUUID()
	})

	It("should support whitelisting IPs", func() {
		By("initializing service from catalog", func() {
			service = helper.NewService(redisServiceID, redisPlanID, []string{"1.1.1.1"})
		})

		By("provisioning a service", func() {
			service.Provision()
		})

		defer By("deprovisioning the service", func() {
			service.Deprovision()
		})

		By("binding a resource to the service", func() {
			binding = service.Bind(appID)
		})

		defer By("unbinding the service", func() {
			service.Unbind(binding.ID)
		})

		By("ensuring that whitelist is set", func() {
			deploymentName, err := broker.MakeInstanceName(service.Cfg.DBPrefix, service.InstanceID)
			Expect(err).NotTo(HaveOccurred())
			deployment, errs := service.ComposeClient.GetDeploymentByName(deploymentName)
			Expect(errs).To(BeEmpty())
			whitelist, errs := service.ComposeClient.GetWhitelistForDeployment(deployment.ID)
			Expect(errs).To(BeEmpty())

			Expect(len(whitelist)).To(Equal(1))
			Expect(whitelist[0].IP).To(Equal("1.1.1.1/32"))
		})

		By("ensuring that access is denied", func() {
			var err error
			conn, err = redis.DialURL(binding.Credentials.URI)
			Expect(err).To(HaveOccurred())
		})

		defer By("disconnecting from the service", func() {
			if conn != nil {
				err := conn.Close()
				Expect(err).ToNot(HaveOccurred())
			}
		})
	})
})
