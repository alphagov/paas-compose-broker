package integration_test

import (
	"github.com/garyburd/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("Broker Compose Integration", func() {

	Context("Redis", func() {

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

		It("should support the full instance lifecycle", func() {

			By("initializing service from catalog", func() {
				service = helper.NewService(redisServiceID, redisPlanID)
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

			By("ensuring binding credentials allow connecting to the service", func() {
				var err error
				conn, err = redis.DialURL(binding.Credentials.URI)
				Expect(err).ToNot(HaveOccurred())
			})

			defer By("disconnecting from the service", func() {
				err := conn.Close()
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring binding credentials allow writing data", func() {
				_, err := conn.Do("SET", "hello", "world")
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring binding credentials allow reading data", func() {
				s, err := redis.String(conn.Do("GET", "hello"))
				Expect(err).ToNot(HaveOccurred())
				Expect(s).To(Equal("world"))
			})

			By("ensuring binding credentials allow deleting data", func() {
				_, err := conn.Do("DEL", "hello")
				Expect(err).ToNot(HaveOccurred())
				ok, _ := redis.Bool(conn.Do("EXISTS", "hello"))
				Expect(ok).To(Equal(false))
			})

		})

	})

})
