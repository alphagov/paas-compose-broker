package integration_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/lager"
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose"
	"github.com/alphagov/paas-compose-broker/compose/fakes"
	"github.com/alphagov/paas-compose-broker/config"
	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration test suite")
}

const (
	instanceCreateTimeout = 15 * time.Minute
	catalogData           = `
{
  "services": [{
    "id": "36f8bf47-c9e7-46d9-880f-5dfc838d05cb",
    "name": "mongodb",
    "description": "Compose MongoDB instance",
    "requires": [],
    "tags": [
      "mongo",
      "compose"
    ],
    "metadata": {
      "displayName": "MongoDB",
      "imageUrl": "https://webassets.mongodb.com/_com_assets/cms/MongoDB-Logo-5c3a7405a85675366beb3a5ec4c032348c390b3f142f5e6dddf1d78e2df5cb5c.png",
      "longDescription": "Compose MongoDB instance",
      "providerDisplayName": "GOV.UK PaaS",
      "documentationUrl": "https://compose.com/mongodb",
      "supportUrl": "https://www.cloud.service.gov.uk/support.html"
    },
    "plans": [{
      "id": "fdfd4fc1-ce69-451c-a436-c2e2795b9abe",
      "name": "small",
      "description": "1GB Storage / 102MB RAM at $35.00/month.",
      "metadata": {
        "displayName": "Mongo Small",
        "bullets": [],
        "units": 1,
        "costs": [{
          "amount": {
            "USD": 35
          },
          "unit": "MONTHLY"
        }]
      }
    }]
  }]
}
`
	serviceID = "36f8bf47-c9e7-46d9-880f-5dfc838d05cb"
	planID    = "fdfd4fc1-ce69-451c-a436-c2e2795b9abe"
)

var (
	newCatalog *catalog.ComposeCatalog
)

var _ = BeforeSuite(func() {
	var err error

	newCatalog, err = catalog.Load(strings.NewReader(catalogData))
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Broker integration tests", func() {
	var (
		cfg            *config.Config
		logger         lager.Logger
		brokerInstance *broker.Broker
		brokerAPI      http.Handler
		composeClient  compose.Client
	)
	BeforeEach(func() {
		cfg = &config.Config{
			Username: randString(10),
			Password: randString(10),
			DBPrefix: "test-suite",
		}

		logger = lager.NewLogger("compose-broker")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, cfg.LogLevel))
	})

	JustBeforeEach(func() {
		var err error
		brokerInstance, err = broker.New(composeClient, cfg, newCatalog, logger)
		Expect(err).NotTo(HaveOccurred())

		brokerAPI = brokerapi.New(brokerInstance, logger, brokerapi.BrokerCredentials{
			Username: cfg.Username,
			Password: cfg.Password,
		})
	})

	Context("Using a mocked Compose API", func() {
		var (
			fakeComposeClient    *fakes.FakeComposeClient
			instanceID           string
			localMongoDBHostname string
			localMongoDBPort     string
			localMongoDBUrl      string
		)

		BeforeEach(func() {
			fakeComposeClient = fakes.New()
			composeClient = fakeComposeClient

			fakeComposeClient.Account = composeapi.Account{ID: "1"}

			localMongoDBHostname = "localhost"
			localMongoDBPort = "27017"
			localMongoDBUrl = fmt.Sprintf("mongodb://%s:%s", localMongoDBHostname, localMongoDBPort)

			instanceID = makeUUID()
			fakeComposeClient.Deployments = []composeapi.Deployment{
				{
					ID:                  "1",
					Name:                fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
					Connection:          composeapi.ConnectionStrings{Direct: []string{localMongoDBUrl}},
					CACertificateBase64: "AAAA",
				},
				{
					ID:         "2",
					Name:       fmt.Sprintf("%s-%s", cfg.DBPrefix, makeUUID()),
					Connection: composeapi.ConnectionStrings{Direct: []string{"mongodb://admin:password@aws-eu-west-1-portal.3.dblayer.com:18899,aws-eu-west-1-portal.8.dblayer.com:18899/admin?ssl=true"}},
				},
			}
		})

		JustBeforeEach(func() {
			brokerInstance.RequireTLS = false
		})

		It("serves the catalog endpoint", func() {
			req := helper.NewRequest("GET", "/v2/catalog", nil, cfg.Username, cfg.Password)
			resp := doRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))

			var returnedCatalog struct {
				Services []struct {
					Name string `json:"name"`
				} `json:"services"`
			}
			err := json.NewDecoder(resp.Body).Decode(&returnedCatalog)
			Expect(err).ToNot(HaveOccurred())
			Expect(returnedCatalog.Services[0].Name).To(Equal("mongodb"))
		})

		Describe("Provisioning an instance", func() {
			var (
				instanceID string
				path       string
			)

			BeforeEach(func() {
				instanceID = makeUUID()
				path = "/v2/service_instances/" + instanceID
			})

			It("provisions an instance", func() {
				provisionDetailsJson := fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": %s
				}`, serviceID, planID, "{}")
				req := helper.NewRequest("PUT", path, strings.NewReader(provisionDetailsJson), cfg.Username, cfg.Password, helper.UriParam{Key: "accepts_incomplete", Value: "true"})
				resp := doRequest(brokerAPI, req)

				Expect(resp.Code).To(Equal(202))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))

				expectedDeploymentParams := composeapi.DeploymentParams{
					Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
					AccountID:    "1",
					Datacenter:   broker.ComposeDatacenter,
					DatabaseType: "mongodb",
					Units:        1,
					SSL:          true,
					ClusterID:    "",
				}
				Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
			})

			It("ignores user provided parameters", func() {
				provisionDetailsJson := fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": {
						"disable_ssl": true,
						"wired_tiger": true,
						"version": "1"
					}
			  	}`, serviceID, planID)
				req := helper.NewRequest("PUT", path, strings.NewReader(provisionDetailsJson), cfg.Username, cfg.Password, helper.UriParam{Key: "accepts_incomplete", Value: "true"})
				resp := doRequest(brokerAPI, req)

				Expect(resp.Code).To(Equal(202))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))

				expectedDeploymentParams := composeapi.DeploymentParams{
					Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
					AccountID:    "1",
					Datacenter:   broker.ComposeDatacenter,
					DatabaseType: "mongodb",
					Units:        1,
					SSL:          true,
					WiredTiger:   false,
					Version:      "",
					ClusterID:    "",
				}
				Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
			})

			Context("when configured with a cluster name", func() {
				BeforeEach(func() {
					cfg.ClusterName = "test-cluster"
					fakeComposeClient.Clusters = []composeapi.Cluster{
						{ID: "1234", Name: "test-cluster"},
					}
				})

				It("provisions in the cluster", func() {

					provisionDetailsJson := fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": %s
				}`, serviceID, planID, "{}")
					req := helper.NewRequest("PUT", path, strings.NewReader(provisionDetailsJson), cfg.Username, cfg.Password, helper.UriParam{Key: "accepts_incomplete", Value: "true"})
					resp := doRequest(brokerAPI, req)

					Expect(resp.Code).To(Equal(202))
					body := helper.ReadResponseBody(resp.Body)
					Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))

					expectedDeploymentParams := composeapi.DeploymentParams{
						Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
						AccountID:    "1",
						Datacenter:   broker.ComposeDatacenter,
						DatabaseType: "mongodb",
						Units:        1,
						SSL:          true,
						ClusterID:    "1234",
					}
					Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
				})
			})
		})

		Describe("deprovisioning an instance", func() {
			It("deprovisions the correct instance", func() {
				path := "/v2/service_instances/" + instanceID
				req := helper.NewRequest(
					"DELETE",
					path,
					nil,
					cfg.Username,
					cfg.Password,
					helper.UriParam{Key: "service_id", Value: serviceID},
					helper.UriParam{Key: "plan_id", Value: planID},
					helper.UriParam{Key: "accepts_incomplete", Value: "true"},
				)
				resp := doRequest(brokerAPI, req)

				Expect(resp.Code).To(Equal(202))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"deprovision-recipe-id\",\"type\":\"deprovision\"}`))

				Expect(fakeComposeClient.DeprovisionDeploymentID).To(Equal("1"))
			})
		})

		Describe("updating a service", func() {
			It("does not allow updating the plan", func() {
				path := fmt.Sprintf("/v2/service_instances/%s", instanceID)
				newPlanID := "Plan-2"
				provisionDetailsJson := fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"previous_values": {
						"plan_id": "%s"
					},
					"parameters": "{}"
				}`, serviceID, newPlanID, planID)
				req := helper.NewRequest(
					"PATCH",
					path,
					strings.NewReader(provisionDetailsJson),
					cfg.Username,
					cfg.Password,
					helper.UriParam{Key: "accepts_incomplete", Value: "true"},
				)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(500))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring("changing plans is not currently supported"))
			})
		})

		Describe("binding to a service", func() {
			It("returns binding information", func() {
				bindingID := makeUUID()
				req := bindRequest(instanceID, bindingID, serviceID, planID, makeUUID(), cfg)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(201))

				var data struct {
					Credentials map[string]string `json:"credentials"`
				}
				err := json.NewDecoder(resp.Body).Decode(&data)
				Expect(err).ToNot(HaveOccurred())

				Expect(data.Credentials["host"]).To(Equal(localMongoDBHostname))
				Expect(data.Credentials["port"]).To(Equal(localMongoDBPort))
				Expect(data.Credentials["name"]).To(Equal("db_" + instanceID))
				Expect(data.Credentials["username"]).To(Equal("user_" + bindingID))
				Expect(data.Credentials["ca_certificate_base64"]).To(Equal("AAAA"))
			})
		})

		Describe("unbinding from a service", func() {
			It("allows unbinding a service", func() {
				bindingID := makeUUID()
				req := bindRequest(instanceID, bindingID, serviceID, planID, makeUUID(), cfg)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(201))

				req = unbindRequest(instanceID, bindingID, serviceID, planID, cfg)
				resp = doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
			})
		})

		Describe("polling for the status of the last operation", func() {
			var (
				path string
				req  *http.Request
			)

			BeforeEach(func() {
				path = fmt.Sprintf("/v2/service_instances/%s/last_operation", makeUUID())
				req = helper.NewRequest(
					"GET",
					path,
					nil,
					cfg.Username,
					cfg.Password,
					helper.UriParam{Key: "service_id", Value: serviceID},
					helper.UriParam{Key: "plan_id", Value: planID},
					helper.UriParam{Key: "operation", Value: "{\"recipe_id\":\"recipe-id\",\"type\":\"provision\"}"},
				)
			})

			It("returns an error when unable to get the recipe", func() {
				fakeComposeClient.GetRecipeErr = fmt.Errorf("error: failed to get recipe by ID")

				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(500))
				Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring(`{"description":"error: failed to get recipe by ID"}`))
			})

			It("returns a failed state when the Compose recipe status is not recognised", func() {
				fakeComposeClient.GetRecipeStatus = "some-unknown-recipe-status"

				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring("failed"))
			})

			It("returns OK when last operation has completed", func() {
				fakeComposeClient.GetRecipeStatus = "complete"

				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring("succeeded"))
			})

			It("returns OK when last operation is still running", func() {
				fakeComposeClient.GetRecipeStatus = "running"

				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring("in progress"))
			})

			It("returns OK when last operation is waiting to run", func() {
				fakeComposeClient.GetRecipeStatus = "waiting"

				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
				body := helper.ReadResponseBody(resp.Body)
				Expect(string(body)).To(ContainSubstring("in progress"))
			})
		})
	})

	Context("Connecting to the real Compose API", func() {
		var (
			instanceID string
		)

		BeforeEach(func() {
			if os.Getenv("SKIP_COMPOSE_API_TESTS") == "true" {
				Skip("SKIP_COMPOSE_API_TESTS is set, skipping tests against real Compose API")
			}

			var err error
			cfg.APIToken = os.Getenv("ACCESS_TOKEN")
			Expect(cfg.APIToken).NotTo(BeEmpty(), "Please export $ACCESS_TOKEN")

			composeClient, err = compose.NewClient(cfg.APIToken)
			Expect(err).NotTo(HaveOccurred())

			instanceID = makeUUID()

			clusters, errs := composeClient.GetClusters()
			Expect(errs).To(BeNil())
			cfg.ClusterName = (*clusters)[0].Name
		})

		AfterEach(func() {
			// Clean-up instance in case of test failures.
			path := "/v2/service_instances/" + instanceID
			req := helper.NewRequest(
				"DELETE",
				path,
				nil,
				cfg.Username,
				cfg.Password,
				helper.UriParam{Key: "service_id", Value: serviceID},
				helper.UriParam{Key: "plan_id", Value: planID},
				helper.UriParam{Key: "accepts_incomplete", Value: "true"},
			)
			doRequest(brokerAPI, req)
		})

		It("supports the full instance lifecycle", func() {

			By("provisioning an instance", func() {
				path := "/v2/service_instances/" + instanceID
				provisionDetailsJson := fmt.Sprintf(`
					{
						"service_id": "%s",
						"plan_id": "%s",
						"organization_guid": "test-organization-id",
						"space_guid": "space-id",
						"parameters": "{}"
					}
				`, serviceID, planID)
				req := helper.NewRequest("PUT", path, strings.NewReader(provisionDetailsJson), cfg.Username, cfg.Password, helper.UriParam{Key: "accepts_incomplete", Value: "true"})
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(202))

				var provisionResp brokerapi.ProvisioningResponse
				err := json.NewDecoder(resp.Body).Decode(&provisionResp)
				Expect(err).ToNot(HaveOccurred())

				operationState := pollForOperationCompletion(
					cfg, brokerAPI,
					instanceID, serviceID, planID, provisionResp.OperationData,
				)
				Expect(operationState).To(Equal("succeeded"), "and returns success")
			})

			bindingID := makeUUID()
			appGUID := makeUUID()
			var bindingData struct {
				Credentials map[string]string `json:"credentials"`
			}
			var rebindingData struct {
				Credentials map[string]string `json:"credentials"`
			}
			By("binding to an instance", func() {
				req := bindRequest(instanceID, bindingID, serviceID, planID, appGUID, cfg)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(201))

				err := json.NewDecoder(resp.Body).Decode(&bindingData)
				Expect(err).ToNot(HaveOccurred())
			})

			By("generating a non-admin user", func() {
				Expect(bindingData.Credentials["username"]).ToNot(Equal("admin"))
			})

			By("using a non-admin database", func() {
				Expect(bindingData.Credentials["name"]).ToNot(Equal("admin"))
			})

			By("connecting and read-writing to the instance", func() {
				session, err := broker.MongoConnection(bindingData.Credentials["uri"], bindingData.Credentials["ca_certificate_base64"], brokerInstance.RequireTLS)
				Expect(err).ToNot(HaveOccurred())
				defer session.Close()

				input := &Person{Name: "John Jones", Phone: "+447777777777"}
				people := session.DB(rebindingData.Credentials["name"]).C("people")
				err = people.Insert(input)
				Expect(err).ToNot(HaveOccurred())

				var result Person
				err = people.Find(bson.M{"name": "John Jones"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Name).To(Equal(input.Name))
				Expect(result.Phone).To(Equal(input.Phone))
			})

			By("being unable to access the admin database", func() {
				session, err := broker.MongoConnection(bindingData.Credentials["uri"], bindingData.Credentials["ca_certificate_base64"], brokerInstance.RequireTLS)
				Expect(err).ToNot(HaveOccurred())
				defer session.Close()

				db := session.DB("admin")
				// Check we can't list the collections in the `admin` database.
				_, err = db.CollectionNames()
				Expect(err).To(HaveOccurred())
				Expect(err.(*mgo.QueryError).Code).To(Equal(13))
				Expect(err.(*mgo.QueryError).Message).To(ContainSubstring("not authorized"))

				// Check we can't write to the `admin` database.
				input := &Person{Name: "Major Kong", Phone: "+1166666666666"}
				bad_people := db.C("bad_people")
				err = bad_people.Insert(input)
				Expect(err).To(HaveOccurred())
				Expect(err.(*mgo.QueryError).Code).To(Equal(13))
				Expect(err.(*mgo.QueryError).Message).To(ContainSubstring("not authorized"))
			})

			By("being unable to modify user permissions", func() {
				session, err := broker.MongoConnection(bindingData.Credentials["uri"], bindingData.Credentials["ca_certificate_base64"], brokerInstance.RequireTLS)
				Expect(err).ToNot(HaveOccurred())
				defer session.Close()

				db := session.DB(rebindingData.Credentials["name"])
				// Ensure we can't make ourselves a global admin.
				err = db.UpsertUser(&mgo.User{
					Username: rebindingData.Credentials["username"],
					Password: rebindingData.Credentials["password"],
					Roles:    []mgo.Role{mgo.RoleDBAdminAny},
				})
				Expect(err).To(HaveOccurred())
			})

			By("being unable to create users", func() {
				session, err := broker.MongoConnection(bindingData.Credentials["uri"], bindingData.Credentials["ca_certificate_base64"], brokerInstance.RequireTLS)
				Expect(err).ToNot(HaveOccurred())
				defer session.Close()

				db := session.DB(rebindingData.Credentials["name"])

				roles_to_try := []mgo.Role{mgo.RoleReadWrite, mgo.RoleDBAdmin, mgo.RoleReadWriteAny, mgo.RoleDBAdminAny}
				for _, role_to_try := range roles_to_try {
					err = db.UpsertUser(&mgo.User{
						Username: "new_user_they_should_not_be_able_to_create_" + string(role_to_try),
						Password: "zoomzoom" + string(role_to_try),
						Roles:    []mgo.Role{role_to_try},
					})
					Expect(err).To(HaveOccurred())
					Expect(err.(*mgo.QueryError).Code).To(Equal(13))
					Expect(err.(*mgo.QueryError).Message).To(ContainSubstring("not authorized"))
				}
			})

			By("being unable to create databases", func() {
				session, err := broker.MongoConnection(bindingData.Credentials["uri"], bindingData.Credentials["ca_certificate_base64"], brokerInstance.RequireTLS)
				Expect(err).ToNot(HaveOccurred())
				defer session.Close()

				// Ensure we can't insert data to another database.
				db := session.DB("a_new_db_they_should_not_be_able_to_use")
				input := &Person{Name: "Major Kong", Phone: "+1166666666666"}
				bad_people := db.C("bad_people")
				err = bad_people.Insert(input)
				Expect(err).To(HaveOccurred())
				Expect(err.(*mgo.QueryError).Code).To(Equal(13))
				Expect(err.(*mgo.QueryError).Message).To(ContainSubstring("not authorized"))
			})

			By("checking if instance uses enterprise cluster", func() {
				deploymentName := fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID)
				deployment, errs := composeClient.GetDeploymentByName(deploymentName)
				Expect(errs).To(BeNil())
				clusterURL, err := url.Parse(deployment.Links.ClusterLink.HREF)
				Expect(err).ToNot(HaveOccurred())
				splitPath := strings.Split(strings.TrimRight(clusterURL.Path, "{"), "/")
				clusterID := splitPath[len(splitPath)-1]
				expectedCluster, errs := composeClient.GetClusterByName(cfg.ClusterName)
				Expect(errs).To(BeNil())
				Expect(clusterID).To(Equal(expectedCluster.ID))
				Expect(expectedCluster.Type).To(Equal("private"))
			})

			By("unbinding from the service", func() {
				req := unbindRequest(instanceID, bindingID, serviceID, planID, cfg)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))

				// Response will be an empty JSON object for future compatibility
				var data map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&data)
				Expect(err).ToNot(HaveOccurred())
			})

			By("rebinding to the service", func() {
				req := bindRequest(instanceID, bindingID, serviceID, planID, appGUID, cfg)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(201))

				err := json.NewDecoder(resp.Body).Decode(&rebindingData)
				Expect(err).ToNot(HaveOccurred())
			})

			By("providing a new password upon rebind", func() {
				Expect(rebindingData.Credentials["username"]).To(Equal(bindingData.Credentials["username"]))
				Expect(rebindingData.Credentials["password"]).ToNot(Equal(bindingData.Credentials["password"]))
			})

			By("using the new credentials to alter existing objects", func() {
				session, err := broker.MongoConnection(rebindingData.Credentials["uri"], rebindingData.Credentials["ca_certificate_base64"], brokerInstance.RequireTLS)
				Expect(err).ToNot(HaveOccurred())
				defer session.Close()

				people := session.DB(rebindingData.Credentials["name"]).C("people")
				var result Person

				// Read the person inserted previously.
				err = people.Find(bson.M{"name": "John Jones"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Phone).To(Equal("+447777777777"))

				// Update the name of the person inserted previously.
				err = people.Update(bson.M{"name": "John Jones"}, bson.M{"$set": bson.M{"name": "Jane Jones"}})
				Expect(err).ToNot(HaveOccurred())
				err = people.Find(bson.M{"name": "Jane Jones"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Phone).To(Equal("+447777777777"))

				// Insert another person.
				input2 := &Person{Name: "Tim Timmis", Phone: "+17734573777"}
				err = people.Insert(input2)
				Expect(err).ToNot(HaveOccurred())
				err = people.Find(bson.M{"name": "Tim Timmis"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Phone).To(Equal("+17734573777"))

				// Delete the people collection.
				err = people.DropCollection()
				Expect(err).ToNot(HaveOccurred())
			})

			By("re-unbinding from the service", func() {
				req := unbindRequest(instanceID, bindingID, serviceID, planID, cfg)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))

				// Response will be an empty JSON object for future compatibility
				var data map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&data)
				Expect(err).ToNot(HaveOccurred())
			})

			By("deprovisioning the instance", func() {
				path := "/v2/service_instances/" + instanceID
				req := helper.NewRequest(
					"DELETE",
					path,
					nil,
					cfg.Username,
					cfg.Password,
					helper.UriParam{Key: "service_id", Value: serviceID},
					helper.UriParam{Key: "plan_id", Value: planID},
					helper.UriParam{Key: "accepts_incomplete", Value: "true"},
				)
				resp := doRequest(brokerAPI, req)
				Expect(resp.Code).To(BeEquivalentTo(202))

				var deprovisionResp brokerapi.DeprovisionResponse
				err := json.NewDecoder(resp.Body).Decode(&deprovisionResp)
				Expect(err).ToNot(HaveOccurred())

				operationState := pollForOperationCompletion(
					cfg, brokerAPI, instanceID,
					serviceID, planID, deprovisionResp.OperationData,
				)
				Expect(operationState).To(Equal("succeeded"), "returns success")
			})
		})
	})
})

type Person struct {
	Name  string
	Phone string
}

func doRequest(server http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func bindRequest(instanceID, bindingID, serviceID, planID, appGUID string, cfg *config.Config) *http.Request {
	path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
	bindingDetailsJson := fmt.Sprintf(`
		{
			"service_id": "%s",
			"plan_id": "%s",
			"bind_resource": {
				"app_guid": "%s"
			},
			"parameters": "{}"
		}`,
		serviceID,
		planID,
		appGUID,
	)
	req := helper.NewRequest(
		"PUT",
		path,
		strings.NewReader(bindingDetailsJson),
		cfg.Username,
		cfg.Password,
	)
	return req
}

func unbindRequest(instanceID, bindingID, serviceID, planID string, cfg *config.Config) *http.Request {
	path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
	req := helper.NewRequest(
		"DELETE",
		path,
		nil,
		cfg.Username,
		cfg.Password,
		helper.UriParam{Key: "service_id", Value: serviceID},
		helper.UriParam{Key: "plan_id", Value: planID},
	)
	return req
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func makeUUID() string {
	return uuid.NewV4().String()
}

func pollForOperationCompletion(cfg *config.Config, brokerAPI http.Handler, instanceID, serviceID, planID, operation string) string {
	var state string

	fmt.Fprint(GinkgoWriter, "Polling for Instance Operation to complete")
	Eventually(
		func() string {
			fmt.Fprint(GinkgoWriter, ".")
			path := fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
			req := helper.NewRequest(
				"GET",
				path,
				nil,
				cfg.Username,
				cfg.Password,
				helper.UriParam{Key: "service_id", Value: serviceID},
				helper.UriParam{Key: "plan_id", Value: planID},
				helper.UriParam{Key: "operation", Value: operation},
			)
			resp := doRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))

			var lastOperation map[string]string
			err := json.NewDecoder(resp.Body).Decode(&lastOperation)
			Expect(err).ToNot(HaveOccurred())
			state = lastOperation["state"]
			return state
		},
		instanceCreateTimeout,
		15*time.Second,
	).Should(
		SatisfyAny(
			Equal("succeeded"),
			Equal("failed"),
		),
	)

	fmt.Fprintf(GinkgoWriter, "done. Final state: %s.\n", state)
	return state
}
