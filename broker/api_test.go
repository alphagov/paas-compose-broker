package broker_test

import (
	"errors"
	"strings"

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
		fakeComposeClient *fakes.FakeClient
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

		buildAPI = func(cfg *config.Config, fakeComposeClient *fakes.FakeClient) http.Handler {
			logger := lager.NewLogger("compose-broker")
			logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, cfg.LogLevel))

			broker, err := broker.New(fakeComposeClient, enginefakes.FakeProvider{}, cfg, &catalog.Catalog{
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

			return brokerapi.New(
				broker,
				logger,
				brokerapi.BrokerCredentials{
					Username: cfg.Username,
					Password: cfg.Password,
				},
			)
		}
	)

	BeforeEach(func() {
		cfg = &config.Config{
			Username: "jeff",
			Password: "j3ffers0n",
			DBPrefix: "test",
			IPWhitelist: []string{
				"1.1.1.1",
				"2.2.2.2",
				"3.3.3.3",
			},
		}
	})

	JustBeforeEach(func() {

		fakeComposeClient = &fakes.FakeClient{}
		fakeComposeClient.GetAccountReturns(&composeapi.Account{ID: "1"}, []error{})

		fakeComposeClient.GetDeploymentsReturns(&[]composeapi.Deployment{
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
		}, []error{})
		fakeComposeClient.GetClusterByNameReturns(&composeapi.Cluster{
			ID: "1234", Name: cfg.ClusterName,
		}, []error{})

		logger := lager.NewLogger("compose-broker")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, cfg.LogLevel))

		brokerAPI = buildAPI(cfg, fakeComposeClient)
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
			fakeComposeClient.CreateDeploymentReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(0, &composeapi.Recipe{ID: "id-for-1.1.1.1", Status: "complete"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(1, &composeapi.Recipe{ID: "id-for-2.2.2.2", Status: "complete"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(2, &composeapi.Recipe{ID: "id-for-3.3.3.3", Status: "complete"}, []error{})

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
			Expect(body).To(MatchOperationJSON(`
			{
			  "recipe_id":"provision-recipe-id",
			  "type":"provision",
			  "whitelist_recipe_ids":["id-for-1.1.1.1","id-for-2.2.2.2","id-for-3.3.3.3"]
			}
			`))

			By("creating the deployment")
			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
				AccountID:    "1",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "fakedb",
				Units:        1,
				SSL:          true,
				ClusterID:    "",
			}
			Expect(fakeComposeClient.CreateDeploymentArgsForCall(0)).To(Equal(expectedDeploymentParams))

			By("adding all whitelist entries")
			var args composeapi.DeploymentWhitelistParams
			_, args = fakeComposeClient.CreateDeploymentWhitelistArgsForCall(0)
			Expect(args).To(Equal(composeapi.DeploymentWhitelistParams{
				IP:          "1.1.1.1",
				Description: "Allow 1.1.1.1 to access deployment",
			}))
			_, args = fakeComposeClient.CreateDeploymentWhitelistArgsForCall(1)
			Expect(args).To(Equal(composeapi.DeploymentWhitelistParams{
				IP:          "2.2.2.2",
				Description: "Allow 2.2.2.2 to access deployment",
			}))
			_, args = fakeComposeClient.CreateDeploymentWhitelistArgsForCall(2)
			Expect(args).To(Equal(composeapi.DeploymentWhitelistParams{
				IP:          "3.3.3.3",
				Description: "Allow 3.3.3.3 to access deployment",
			}))

			By("not deprovisioning, as would happen in an error situation")
			Expect(fakeComposeClient.DeprovisionDeploymentCallCount()).
				To(Equal(0))
		})

		It("500s and deprovisions if any of the whitelist entry requests fail", func() {
			instanceID := uuid.NewV4().String()
			fakeComposeClient.CreateDeploymentReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})

			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(
				0, nil, []error{},
			)
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(
				1, &composeapi.Recipe{}, []error{errors.New("won't get here")},
			)

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
			body := ReadResponseBody(resp.Body)

			Expect(resp.Code).To(Equal(500))
			Expect(string(body)).
				To(MatchJSON(`{"description":"malformed response from Compose: no pending whitelist recipe received"}`))
			Expect(fakeComposeClient.DeprovisionDeploymentArgsForCall(0)).
				To(Equal("1"))
		})

		It("500s if any of the whitelist recipes are nil", func() {
			fakeComposeClient.CreateDeploymentReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturns(nil, []error{})

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
			body := ReadResponseBody(resp.Body)

			Expect(resp.Code).To(Equal(500))
			Expect(string(body)).
				To(MatchJSON(`{"description":"malformed response from Compose: no pending whitelist recipe received"}`))
		})

		It("ignores user provided parameters", func() {
			instanceID := uuid.NewV4().String()
			fakeComposeClient.CreateDeploymentReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(0, &composeapi.Recipe{ID: "id-for-1.1.1.1", Status: "complete"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(1, &composeapi.Recipe{ID: "id-for-2.2.2.2", Status: "complete"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(2, &composeapi.Recipe{ID: "id-for-3.3.3.3", Status: "complete"}, []error{})

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
			Expect(body).To(MatchOperationJSON(`
			{
			  "recipe_id":"provision-recipe-id",
			  "type":"provision",
			  "whitelist_recipe_ids":["id-for-1.1.1.1","id-for-2.2.2.2","id-for-3.3.3.3"]
			}
			`))

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
			Expect(fakeComposeClient.CreateDeploymentArgsForCall(0)).To(Equal(expectedDeploymentParams))
		})
	})

	Describe("Provisioning an instance into cluster", func() {

		BeforeEach(func() {
			cfg.ClusterName = "test-cluster"
		})

		It("provisions into cluster when configured with a cluster name", func() {
			instanceID := uuid.NewV4().String()
			fakeComposeClient.CreateDeploymentReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(0, &composeapi.Recipe{ID: "id-for-1.1.1.1", Status: "complete"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(1, &composeapi.Recipe{ID: "id-for-2.2.2.2", Status: "complete"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(2, &composeapi.Recipe{ID: "id-for-3.3.3.3", Status: "complete"}, []error{})

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
			Expect(body).To(MatchOperationJSON(`
			{
			  "recipe_id":"provision-recipe-id",
			  "type":"provision",
			  "whitelist_recipe_ids":["id-for-1.1.1.1","id-for-2.2.2.2","id-for-3.3.3.3"]
			}
			`))

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", cfg.DBPrefix, instanceID),
				AccountID:    "1",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "fakedb",
				Units:        1,
				SSL:          true,
				ClusterID:    "1234",
			}
			Expect(fakeComposeClient.CreateDeploymentArgsForCall(0)).To(Equal(expectedDeploymentParams))
		})

		It("responds with 500 when a whitelist recipe ID is invalid", func() {
			instanceID := uuid.NewV4().String()
			invalid := ""
			fakeComposeClient.CreateDeploymentReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(
				0, &composeapi.Recipe{ID: "id-for-1.1.1.1", Status: "complete"}, []error{},
			)
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(
				1, &composeapi.Recipe{ID: invalid, Status: "complete"}, []error{},
			)
			fakeComposeClient.CreateDeploymentWhitelistReturnsOnCall(
				2, &composeapi.Recipe{ID: "id-for-3.3.3.3", Status: "complete"}, []error{},
			)

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

			Expect(resp.Code).To(Equal(500))
			body := ReadResponseBody(resp.Body)
			Expect(body).To(MatchJSON(`{"description":"malformed response from Compose: invalid whitelist recipe ID"}`))
		})
	})

	Describe("deprovisioning an instance", func() {

		It("deprovisions the correct instance", func() {
			instanceID := uuid.NewV4().String()
			fakeComposeClient.GetDeploymentByNameReturns(&composeapi.Deployment{ID: "1", ProvisionRecipeID: "provision-recipe-id"}, []error{})
			fakeComposeClient.DeprovisionDeploymentReturns(&composeapi.Recipe{ID: "deprovision-recipe-id"}, []error{})
			resp := DoRequest(brokerAPI, NewRequest(
				"DELETE",
				"/v2/service_instances/"+instanceID,
				nil,
				cfg.Username,
				cfg.Password,
				UriParam{Key: "accepts_incomplete", Value: "true"},
			))
			Expect(fakeComposeClient.DeprovisionDeploymentArgsForCall(0)).To(Equal("1"))
			Expect(resp.Code).To(Equal(202))
			Expect(ReadResponseBody(resp.Body)).To(MatchOperationJSON(`{"type":"deprovision","recipe_id":"deprovision-recipe-id", "whitelist_recipe_ids": []}`))
		})

	})

	Describe("updating a service", func() {

		It("does not allow updating the plan", func() {
			fakeComposeClient.GetDeploymentByNameReturns(&composeapi.Deployment{}, []error{})
			oldPlanID := service.Plans[0].ID
			newPlanID := service.Plans[1].ID
			resp := DoRequest(brokerAPI, NewRequest(
				"PATCH",
				"/v2/service_instances/update-me",
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
			Expect(body).To(MatchJSON(`{"description":"changing plans is not currently supported"}`))
		})

	})

	Describe("binding to a service", func() {

		It("returns binding information", func() {
			instanceID := "bind-me-up"
			connectionStrings := composeapi.ConnectionStrings{
				Direct: []string{"i", "just", "love", "connecting"},
			}
			fakeComposeClient.GetDeploymentByNameReturns(&composeapi.Deployment{
				Type:       "fakedb",
				Connection: connectionStrings,
			}, []error{})

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
				Credentials map[string]interface{} `json:"credentials"`
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
			instanceID := "please-unbind-me"
			bindingID := uuid.NewV4().String()
			appID := uuid.NewV4().String()
			connectionStrings := composeapi.ConnectionStrings{
				Direct: []string{"i", "just", "love", "connecting"},
			}

			fakeComposeClient.GetDeploymentByNameReturns(&composeapi.Deployment{
				Type:       "fakedb",
				Connection: connectionStrings,
			}, []error{})

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
		Context("without whitelist", func() {
			var req *http.Request

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
				fakeComposeClient.GetRecipeReturns(
					nil,
					[]error{fmt.Errorf("error: failed to get recipe by ID")},
				)

				resp := DoRequest(brokerAPI, req)

				Expect(fakeComposeClient.GetRecipeArgsForCall(0)).To(Equal("recipe-id"))
				Expect(resp.Code).To(Equal(500))
				Expect(ReadResponseBody(resp.Body)).
					To(MatchJSON(`{"description":"error: failed to get recipe by ID"}`))
			})

			It("returns a failed state when the Compose recipe status is not recognised", func() {
				fakeComposeClient.GetRecipeReturns(
					&composeapi.Recipe{Status: "some-unknown-recipe-status"},
					[]error{},
				)

				resp := DoRequest(brokerAPI, req)

				Expect(fakeComposeClient.GetRecipeArgsForCall(0)).To(Equal("recipe-id"))
				Expect(resp.Code).To(Equal(200))
				Expect(ReadResponseBody(resp.Body)).
					To(MatchJSON(`{"state":"failed"}`))
			})

			It("returns OK when last operation has completed", func() {
				fakeComposeClient.GetRecipeReturns(
					&composeapi.Recipe{Status: "complete"},
					[]error{},
				)

				resp := DoRequest(brokerAPI, req)

				Expect(fakeComposeClient.GetRecipeArgsForCall(0)).To(Equal("recipe-id"))
				Expect(resp.Code).To(Equal(200))
				Expect(ReadResponseBody(resp.Body)).To(MatchJSON(`{"state":"succeeded"}`))
			})

			It("returns OK when last operation is still running", func() {
				fakeComposeClient.GetRecipeReturns(
					&composeapi.Recipe{Status: "running"},
					[]error{},
				)

				resp := DoRequest(brokerAPI, req)

				Expect(fakeComposeClient.GetRecipeArgsForCall(0)).To(Equal("recipe-id"))
				Expect(resp.Code).To(Equal(200))
				Expect(ReadResponseBody(resp.Body)).To(MatchJSON(`{"state":"in progress"}`))
			})

			It("returns OK when last operation is waiting to run", func() {
				fakeComposeClient.GetRecipeReturns(
					&composeapi.Recipe{Status: "waiting"},
					[]error{},
				)

				resp := DoRequest(brokerAPI, req)

				Expect(fakeComposeClient.GetRecipeArgsForCall(0)).To(Equal("recipe-id"))
				Expect(resp.Code).To(Equal(200))
				Expect(ReadResponseBody(resp.Body)).To(MatchJSON(`{"state":"in progress"}`))
			})

			It("responds OK on success", func() {
				fakeComposeClient.GetRecipeReturns(&composeapi.Recipe{Status: "complete"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"succeeded"}`))
			})

			It("responds OK on failure to deploy", func() {
				fakeComposeClient.GetRecipeReturns(&composeapi.Recipe{Status: "some-failure-state"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"failed"}`))
			})

			It("responds with Error when deployment status isn't available", func() {
				fakeComposeClient.GetRecipeReturns(&composeapi.Recipe{}, []error{fmt.Errorf("some-error")})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(500))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"description":"some-error"}`))
			})
		})

		Context("with whitelist", func() {
			var req *http.Request

			BeforeEach(func() {
				brokerAPI = buildAPI(&config.Config{
					Username: "jeff",
					Password: "j3ffers0n",
					DBPrefix: "test",
					IPWhitelist: []string{
						"1.1.1.1",
						"2.2.2.2",
						"3.3.3.3",
					},
				}, fakeComposeClient)

				req = NewRequest(
					"GET",
					fmt.Sprintf("/v2/service_instances/%s/last_operation", uuid.NewV4().String()),
					nil,
					cfg.Username,
					cfg.Password,
					UriParam{Key: "service_id", Value: service.ID},
					UriParam{Key: "plan_id", Value: service.Plans[0].ID},
					UriParam{Key: "operation", Value: `
						{
							"type": "provision",
							"recipe_id": "recipe-id-1",
							"whitelist_recipe_ids": [
								"recipe-id-2",
								"recipe-id-3",
								"recipe-id-4"
							]
						}
					`},
				)
			})

			It("responds OK when all successful", func() {
				fakeComposeClient.GetRecipeReturnsOnCall(0, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(1, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(2, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(3, &composeapi.Recipe{Status: "complete"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"succeeded"}`))
			})

			It("responds OK with a failed state when any whitelist entry fails to create", func() {
				fakeComposeClient.GetRecipeReturnsOnCall(0, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(1, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(2, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(3, &composeapi.Recipe{Status: "some-failure-state"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"failed"}`))
			})

			It("responds OK with a failed state when deployment fails and whitelists are pending", func() {
				fakeComposeClient.GetRecipeReturnsOnCall(0, &composeapi.Recipe{Status: "failed"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(1, &composeapi.Recipe{Status: "waiting"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(2, &composeapi.Recipe{Status: "waiting"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(3, &composeapi.Recipe{Status: "waiting"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"failed"}`))
			})

			It("responds OK with an 'in progress' state when any whitelist recipe is running", func() {
				fakeComposeClient.GetRecipeReturnsOnCall(0, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(1, &composeapi.Recipe{Status: "running"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(2, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(3, &composeapi.Recipe{Status: "complete"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"in progress"}`))
			})

			It("responds OK with an 'in progress' state when any whitelist recipe is waiting", func() {
				fakeComposeClient.GetRecipeReturnsOnCall(0, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(1, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(2, &composeapi.Recipe{Status: "waiting"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(3, &composeapi.Recipe{Status: "complete"}, []error{})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(200))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"state":"in progress"}`))
			})

			It("responds with Error when failing to get whitelist status", func() {
				fakeComposeClient.GetRecipeReturnsOnCall(0, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(1, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(2, &composeapi.Recipe{Status: "complete"}, []error{})
				fakeComposeClient.GetRecipeReturnsOnCall(3, &composeapi.Recipe{}, []error{fmt.Errorf("some-error")})
				resp := DoRequest(brokerAPI, req)
				Expect(resp.Code).To(Equal(500))
				body := ReadResponseBody(resp.Body)
				Expect(body).To(MatchJSON(`{"description":"some-error"}`))
			})
		})
	})
})
