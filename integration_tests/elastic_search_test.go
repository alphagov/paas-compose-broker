package integration_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"strings"

	"github.com/alphagov/paas-compose-broker/client/elastic"
	"github.com/alphagov/paas-compose-broker/dbengine"
	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("Broker Compose Integration", func() {

	Context("ElasticSearch", func() {

		const (
			elasticSearchServiceID = "6e9202f2-c2e1-4de8-8d4a-a8c898fc2d8c"
			elasticSearchPlanID    = "6d051078-0913-403c-9763-1d03ecee50d9"
		)

		var (
			service *helper.ServiceHelper
			binding *helper.BindingData
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
				service = helper.NewService(elasticSearchServiceID, elasticSearchPlanID, []string{})
			})

			By("provisioning the service", func() {
				service.Provision()
			})

			defer By("deprovisoning the service", func() {
				service.Deprovision()
			})

			By("binding a resource to service", func() {
				binding = service.Bind(appID)
			})

			defer By("unbinding the service", func() {
				service.Unbind(binding.ID)
			})

			By("connecting to the service")
			httpClient, err := dbengine.SetupHTTPClient(binding.Credentials.CACertificateBase64)
			Expect(err).NotTo(HaveOccurred())

			client, err := elastic.New(binding.Credentials.URI, httpClient)
			Expect(err).NotTo(HaveOccurred())

			version, err := client.Version()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).NotTo(BeEmpty())

			By("ensuring credentials allow writing data")
			putURI := binding.Credentials.URI + "twitter/tweet/1?op_type=create"
			putData := "{\"user\" : \"kimchy\",\"post_date\" : \"2009-11-15T14:12:12\",\"message\" : \"trying out Elasticsearch\"}"
			request, err := http.NewRequest("PUT", putURI, strings.NewReader(putData))
			Expect(err).NotTo(HaveOccurred())
			request.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(request)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(201))

			By("ensuring credentials allow reading data")
			getURI := binding.Credentials.URI + "twitter/tweet/1"
			get, err := client.Get(getURI)
			Expect(err).NotTo(HaveOccurred())
			Expect(get.StatusCode).To(Equal(200))
			body, err := ioutil.ReadAll(get.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring(putData))
		})

	})

})
