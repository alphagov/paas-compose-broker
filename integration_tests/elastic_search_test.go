package integration_test

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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
			putData                = "{\"user\" : \"kimchy\",\"post_date\" : \"2009-11-15T14:12:12\",\"message\" : \"trying out Elasticsearch\"}"
		)

		var (
			service                 *helper.ServiceHelper
			instanceID, instanceID2 string
			binding, binding2       *helper.BindingData
			appID                   string
			client, client2         *elastic.Client
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
				instanceID = service.Provision(map[string]interface{}{})
			})

			defer By("deprovisoning the service", func() {
				service.Deprovision(instanceID)
			})

			By("binding a resource to service", func() {
				binding = service.Bind(instanceID, appID)
			})

			defer By("unbinding the service", func() {
				service.Unbind(instanceID, binding.ID)
			})

			By("connecting to the service", func() {
				httpClient, err := dbengine.SetupHTTPClient(binding.Credentials.CACertificateBase64)
				Expect(err).NotTo(HaveOccurred())

				client, err = elastic.New(binding.Credentials.URI, httpClient)
				Expect(err).NotTo(HaveOccurred())

				version, err := client.Version()
				Expect(err).NotTo(HaveOccurred())
				Expect(version).NotTo(BeEmpty())
			})

			By("ensuring credentials allow writing data", func() {
				putURI := binding.Credentials.URI + "twitter/tweet/1?op_type=create"
				request, err := http.NewRequest("PUT", putURI, strings.NewReader(putData))
				Expect(err).NotTo(HaveOccurred())
				request.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(request)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(201))
			})

			By("ensuring credentials allow reading data", func() {
				getURI := binding.Credentials.URI + "twitter/tweet/1"
				get, err := client.Get(getURI)
				Expect(err).NotTo(HaveOccurred())
				Expect(get.StatusCode).To(Equal(200))
				body, err := ioutil.ReadAll(get.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(ContainSubstring(putData))
			})

			By("ensuring credentials allow writing data in each URI", func() {
				for idx, uri := range binding.Credentials.URIs {
					tweet_id := idx + 2

					By("ensuring credentials allow writing data to URI " + strconv.Itoa(idx))
					putURI := uri + "twitter/tweet/" + strconv.Itoa(tweet_id) + "?op_type=create"
					putData := "{\"user\" : \"kimchy\",\"post_date\" : \"2009-11-15T14:12:12\",\"message\" : \"trying out Elasticsearch multiple uris\"}"
					request, err := http.NewRequest("PUT", putURI, strings.NewReader(putData))
					Expect(err).NotTo(HaveOccurred())
					request.Header.Set("Content-Type", "application/json")
					resp, err := client.Do(request)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(201))

					By("ensuring credentials allow reading data from URI " + strconv.Itoa(idx))
					getURI := uri + "twitter/tweet/" + strconv.Itoa(tweet_id)
					get, err := client.Get(getURI)
					Expect(err).NotTo(HaveOccurred())
					Expect(get.StatusCode).To(Equal(200))
					body, err := ioutil.ReadAll(get.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(body).To(ContainSubstring(putData))
				}
			})

			By("ensuring we have a backup", func() {
				deploymentName := fmt.Sprintf("%s-%s", service.Cfg.DBPrefix, instanceID)
				deployment, errs := service.ComposeClient.GetDeploymentByName(deploymentName)
				Expect(errs).To(BeNil())
				recipe, errs := service.ComposeClient.StartBackupForDeployment(deployment.ID)
				Expect(errs).To(BeNil())
				Eventually(func() bool {
					recipe, err := service.ComposeClient.GetRecipe(recipe.ID)
					return err == nil && recipe.Status == "complete"
				}, 15*time.Minute, 30*time.Second).Should(BeTrue())
			})

			By("creating a new service instance from backup", func() {
				instanceID2 = service.Provision(map[string]interface{}{"restore_from_latest_snapshot_of": instanceID})
			})

			defer By("deprovisioning the service created from backup", func() {
				service.Deprovision(instanceID2)
			})

			By("binding the app to the service created from backup", func() {
				binding2 = service.Bind(instanceID2, appID)
			})

			defer By("unbinding the service created from backup", func() {
				service.Unbind(instanceID2, binding2.ID)
			})

			By("connecting to the service created from backup", func() {
				httpClient, err := dbengine.SetupHTTPClient(binding2.Credentials.CACertificateBase64)
				Expect(err).NotTo(HaveOccurred())

				client2, err = elastic.New(binding2.Credentials.URI, httpClient)
				Expect(err).NotTo(HaveOccurred())

				version, err := client2.Version()
				Expect(err).NotTo(HaveOccurred())
				Expect(version).NotTo(BeEmpty())
			})

			By("ensuring the service created from backup contains the data from the other instance", func() {
				getURI := binding2.Credentials.URI + "twitter/tweet/1"
				get, err := client2.Get(getURI)
				Expect(err).NotTo(HaveOccurred())
				Expect(get.StatusCode).To(Equal(200))
				body, err := ioutil.ReadAll(get.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body).To(ContainSubstring(putData))
			})
		})

	})

})
