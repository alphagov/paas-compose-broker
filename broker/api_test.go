package broker_test

import (
	"code.cloudfoundry.org/lager"
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose/fakes"
	"github.com/alphagov/paas-compose-broker/config"
	enginefakes "github.com/alphagov/paas-compose-broker/dbengine/fakes"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"
)

func NewRequest(method, path string, body io.Reader, username, password string, params ...UriParam) *http.Request {
	brokerUrl := fmt.Sprintf("http://%s", "127.0.0.1:8080"+path)
	req := httptest.NewRequest(method, brokerUrl, body)
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	q := req.URL.Query()
	for _, p := range params {
		q.Add(p.Key, p.Value)
	}
	req.URL.RawQuery = q.Encode()
	return req
}

type UriParam struct {
	Key   string
	Value string
}

func ReadResponseBody(responseBody *bytes.Buffer) []byte {
	body, err := ioutil.ReadAll(responseBody)
	Expect(err).ToNot(HaveOccurred())
	return body
}

func DoRequest(server http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

var _ = Describe("Broker API", func() {

	var (
		fakeComposeClient *fakes.FakeComposeClient
		cfg               *config.Config
		brokerAPI         http.Handler
		service           = brokerapi.Service{
			ID:            uuid.NewV4().String(),
			Name:          "fakedb",
			Bindable:      true,
			PlanUpdatable: false,
			Plans: []brokerapi.ServicePlan{
				brokerapi.ServicePlan{
					ID:       uuid.NewV4().String(),
					Name:     "fake-plan-1",
					Free:     brokerapi.FreeValue(false),
					Bindable: brokerapi.BindableValue(true),
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "Tiny Fake Plan",
						Costs: []brokerapi.ServicePlanCost{
							brokerapi.ServicePlanCost{
								Amount: map[string]float64{
									"USD": 35,
								},
								Unit: "1",
							},
						},
					},
				},
				brokerapi.ServicePlan{
					ID:       uuid.NewV4().String(),
					Name:     "fake-plan-2",
					Free:     brokerapi.FreeValue(false),
					Bindable: brokerapi.BindableValue(true),
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "Big Fake Plan",
						Costs: []brokerapi.ServicePlanCost{
							brokerapi.ServicePlanCost{
								Amount: map[string]float64{
									"USD": 70,
								},
								Unit: "1",
							},
						},
					},
				},
			},
			Metadata: &brokerapi.ServiceMetadata{
				DisplayName: "MongoDB",
			},
		}
	)

	BeforeEach(func() {
		cfg = &config.Config{
			Username: "jeff",
			Password: "j3ffers0n",
			DBPrefix: "test",
		}
	})

	JustBeforeEach(func() {

		fakeComposeClient = fakes.New()
		fakeComposeClient.Account = composeapi.Account{ID: "1"}
		fakeComposeClient.Deployments = []composeapi.Deployment{
			{
				ID:                  "1111",
				Name:                fmt.Sprintf("%s-%s", cfg.DBPrefix, "1111"),
				Connection:          composeapi.ConnectionStrings{Direct: []string{"fakedb://user:password@localhost:27017/db_1111?ssl=true"}},
				CACertificateBase64: "AAAA",
				Type:                "fakedb",
			},
			{
				ID:         "2222",
				Name:       fmt.Sprintf("%s-%s", cfg.DBPrefix, "2222"),
				Connection: composeapi.ConnectionStrings{Direct: []string{"fakedb://admin:password@host.com:1445/admin?ssl=true"}},
				Type:       "fakedb",
			},
		}
		fakeComposeClient.Clusters = []composeapi.Cluster{
			{ID: "1234", Name: cfg.ClusterName},
		}

		logger := lager.NewLogger("compose-broker")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, cfg.LogLevel))

		fakeDBProvider := enginefakes.FakeProvider{}

		broker, err := broker.New(fakeComposeClient, fakeDBProvider, cfg, &catalog.Catalog{
			Services: []*catalog.Service{
				{
					Plans: []*catalog.Plan{
						{
							ServicePlan: brokerapi.ServicePlan{
								ID: service.Plans[0].ID,
							},
							Compose: catalog.ComposeConfig{
								Units:        1,
								DatabaseType: "fakedb",
							},
						},
					},
					Service: service,
				},
			},
		}, logger)
		Expect(err).NotTo(HaveOccurred())

		brokerAPI = brokerapi.New(
			broker,
			logger,
			brokerapi.BrokerCredentials{
				Username: cfg.Username,
				Password: cfg.Password,
			},
		)
	})

	Describe("Fetching the catalog", func() {

		It("serves the catalog endpoint", func() {
			resp := DoRequest(brokerAPI, NewRequest(
				"GET",
				"/v2/catalog",
				nil,
				cfg.Username,
				cfg.Password,
			))
			Expect(resp.Code).To(Equal(200))

			var returnedCatalog struct {
				Services []struct {
					Name string `json:"name"`
				} `json:"services"`
			}
			err := json.NewDecoder(resp.Body).Decode(&returnedCatalog)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(returnedCatalog.Services)).To(Equal(1))
			Expect(returnedCatalog.Services[0].Name).To(Equal("fakedb"))
		})
	})

	Describe("Provisioning an instance", func() {

		It("provisions an instance", func() {
			instanceID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": {}
				}`, service.ID, service.Plans[0].ID)),
				cfg.Username,
				cfg.Password,
				UriParam{Key: "accepts_incomplete", Value: "true"},
			))
			Expect(resp.Code).To(Equal(202))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
				AccountID:    "1",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "fakedb",
				Units:        1,
				SSL:          true,
				ClusterID:    "",
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
		})

		It("ignores user provided parameters", func() {
			instanceID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
						"service_id": "%s",
						"plan_id": "%s",
						"organization_guid": "test-organization-id",
						"space_guid": "space-id",
						"parameters": {
							"disable_ssl": true,
							"wired_tiger": true,
							"version": "1"
						}
					}`, service.ID, service.Plans[0].ID)),
				cfg.Username,
				cfg.Password,
				UriParam{Key: "accepts_incomplete", Value: "true"},
			))
			Expect(resp.Code).To(Equal(202))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
				AccountID:    "1",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "fakedb",
				Units:        1,
				SSL:          true,
				WiredTiger:   false,
				Version:      "",
				ClusterID:    "",
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
		})
	})

	Describe("Provisioning an instance into cluster", func() {

		BeforeEach(func() {
			cfg.ClusterName = "test-cluster"
		})

		It("provisions into cluster when configured with a cluster name", func() {
			instanceID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": %s
				}`, service.ID, service.Plans[0].ID, "{}")),
				cfg.Username,
				cfg.Password,
				UriParam{Key: "accepts_incomplete", Value: "true"},
			))

			Expect(resp.Code).To(Equal(202))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
				AccountID:    "1",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "fakedb",
				Units:        1,
				SSL:          true,
				ClusterID:    "1234",
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
		})
	})

	Describe("deprovisioning an instance", func() {

		It("deprovisions the correct instance", func() {
			instanceID := fakeComposeClient.Deployments[0].ID
			resp := DoRequest(brokerAPI, NewRequest(
				"DELETE",
				"/v2/service_instances/"+instanceID,
				nil,
				cfg.Username,
				cfg.Password,
				UriParam{Key: "accepts_incomplete", Value: "true"},
			))
			Expect(resp.Code).To(Equal(202))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"deprovision-recipe-id\",\"type\":\"deprovision\"}`))
			Expect(fakeComposeClient.DeprovisionDeploymentID).To(Equal(instanceID))
		})

	})

	Describe("updating a service", func() {

		It("does not allow updating the plan", func() {
			instanceID := fakeComposeClient.Deployments[0].ID
			oldPlanID := service.Plans[0].ID
			newPlanID := service.Plans[1].ID
			resp := DoRequest(brokerAPI, NewRequest(
				"PATCH",
				"/v2/service_instances/"+instanceID,
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"previous_values": {
						"plan_id": "%s"
					},
					"parameters": "{}"
				}`, service.ID, newPlanID, oldPlanID)),
				cfg.Username,
				cfg.Password,
				UriParam{Key: "accepts_incomplete", Value: "true"},
			))
			Expect(resp.Code).To(Equal(500))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring("changing plans is not currently supported"))
		})

	})

	Describe("binding to a service", func() {

		It("returns binding information", func() {
			instanceID := fakeComposeClient.Deployments[0].ID
			bindingID := uuid.NewV4().String()
			appID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID),
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"bind_resource": {
						"app_guid": "%s"
					},
					"parameters": "{}"
				}`, service.ID, service.Plans[0].ID, appID)),
				cfg.Username,
				cfg.Password,
			))
			Expect(resp.Code).To(Equal(201))
			var data struct {
				Credentials map[string]string `json:"credentials"`
			}
			err := json.NewDecoder(resp.Body).Decode(&data)
			Expect(err).ToNot(HaveOccurred())
			Expect(data.Credentials["host"]).To(Equal("localhost"))
			Expect(data.Credentials["port"]).To(Equal("27017"))
			Expect(data.Credentials["name"]).To(Equal("db_1111"))
			Expect(data.Credentials["username"]).To(Equal("user"))
			Expect(data.Credentials["ca_certificate_base64"]).To(Equal("AAAA"))
		})

	})

	Describe("unbinding from a service", func() {

		It("allows unbinding a service", func() {
			instanceID := fakeComposeClient.Deployments[0].ID
			bindingID := uuid.NewV4().String()
			appID := uuid.NewV4().String()
			resp := DoRequest(brokerAPI, NewRequest(
				"PUT",
				fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID),
				strings.NewReader(fmt.Sprintf(`{
					"service_id": "%s",
					"plan_id": "%s",
					"bind_resource": {
						"app_guid": "%s"
					},
					"parameters": "{}"
				}`, service.ID, service.Plans[0].ID, appID)),
				cfg.Username,
				cfg.Password,
			))
			Expect(resp.Code).To(Equal(201))
			resp = DoRequest(brokerAPI, NewRequest(
				"DELETE",
				fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID),
				nil,
				cfg.Username,
				cfg.Password,
			))
			Expect(resp.Code).To(Equal(200))
		})

	})

	Describe("polling for the status of the last operation", func() {

		var (
			req *http.Request
		)

		BeforeEach(func() {
			req = NewRequest(
				"GET",
				fmt.Sprintf("/v2/service_instances/%s/last_operation", uuid.NewV4().String()),
				nil,
				cfg.Username,
				cfg.Password,
				UriParam{Key: "service_id", Value: service.ID},
				UriParam{Key: "plan_id", Value: service.Plans[0].ID},
				UriParam{Key: "operation", Value: "{\"recipe_id\":\"recipe-id\",\"type\":\"provision\"}"},
			)
		})

		It("returns an error when unable to get the recipe", func() {
			fakeComposeClient.GetRecipeErr = fmt.Errorf("error: failed to get recipe by ID")
			resp := DoRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(500))
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring(`{"description":"error: failed to get recipe by ID"}`))
		})

		It("returns a failed state when the Compose recipe status is not recognised", func() {
			fakeComposeClient.GetRecipeStatus = "some-unknown-recipe-status"
			resp := DoRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring("failed"))
		})

		It("returns OK when last operation has completed", func() {
			fakeComposeClient.GetRecipeStatus = "complete"
			resp := DoRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring("succeeded"))
		})

		It("returns OK when last operation is still running", func() {
			fakeComposeClient.GetRecipeStatus = "running"
			resp := DoRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring("in progress"))
		})

		It("returns OK when last operation is waiting to run", func() {
			fakeComposeClient.GetRecipeStatus = "waiting"
			resp := DoRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			body := ReadResponseBody(resp.Body)
			Expect(string(body)).To(ContainSubstring("in progress"))
		})

	})
})
