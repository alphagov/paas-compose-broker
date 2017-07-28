package helper

import (
	"bytes"
	"code.cloudfoundry.org/lager"
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose"
	"github.com/alphagov/paas-compose-broker/config"

	uuid "github.com/satori/go.uuid"
)

const (
	instanceCreateTimeout = 15 * time.Minute
	pollInterval          = 15 * time.Second
)

type UriParam struct {
	Key   string
	Value string
}

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

func ReadResponseBody(responseBody *bytes.Buffer) []byte {
	body, err := ioutil.ReadAll(responseBody)
	Expect(err).ToNot(HaveOccurred())
	return body
}

func NewUUID() string {
	return uuid.NewV4().String()
}

func PollForOperationCompletion(cfg *config.Config, brokerAPI http.Handler, instanceID, serviceID, planID, operation string) string {
	var state string

	fmt.Fprint(GinkgoWriter, "Polling for Instance Operation to complete")
	Eventually(
		func() string {
			fmt.Fprint(GinkgoWriter, ".")
			path := fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
			req := NewRequest(
				"GET",
				path,
				nil,
				cfg.Username,
				cfg.Password,
				UriParam{Key: "service_id", Value: serviceID},
				UriParam{Key: "plan_id", Value: planID},
				UriParam{Key: "operation", Value: operation},
			)
			resp := DoRequest(brokerAPI, req)
			Expect(resp.Code).To(Equal(200))

			var lastOperation map[string]string
			err := json.NewDecoder(resp.Body).Decode(&lastOperation)
			Expect(err).ToNot(HaveOccurred())
			state = lastOperation["state"]
			return state
		},
		instanceCreateTimeout,
		pollInterval,
	).Should(
		SatisfyAny(
			Equal("succeeded"),
			Equal("failed"),
		),
	)

	fmt.Fprintf(GinkgoWriter, "done. Final state: %s.\n", state)
	return state
}

func DoRequest(server http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

type BindingData struct {
	ID          string
	AppID       string
	Credentials struct {
		Name                string `json:"name"`
		Username            string `json:"username"`
		Password            string `json:"password"`
		URI                 string `json:"uri"`
		CACertificateBase64 string `json:"ca_certificate_base64"`
	} `json:"credentials"`
}

type ServiceHelper struct {
	InstanceID     string
	ServiceID      string
	PlanID         string
	Catalog        *catalog.ComposeCatalog
	Cfg            *config.Config
	Logger         lager.Logger
	BrokerInstance *broker.Broker
	BrokerAPI      http.Handler
	ComposeClient  compose.Client
}

func (s *ServiceHelper) Bind() (binding *BindingData) {
	binding = &BindingData{
		ID:    NewUUID(),
		AppID: NewUUID(),
	}
	resp := DoRequest(s.BrokerAPI, NewRequest(
		"PUT",
		fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", s.InstanceID, binding.ID),
		strings.NewReader(fmt.Sprintf(`{
			"service_id": "%s",
			"plan_id": "%s",
			"bind_resource": {
				"app_guid": "%s"
			},
			"parameters": "{}"
		}`, s.ServiceID, s.PlanID, binding.AppID)),
		s.Cfg.Username,
		s.Cfg.Password,
	))
	Expect(resp.Code).To(Equal(201))

	err := json.NewDecoder(resp.Body).Decode(&binding)
	Expect(err).ToNot(HaveOccurred())

	Expect(binding.Credentials.URI).NotTo(BeEmpty())
	Expect(binding.Credentials.Username).NotTo(BeEmpty())
	Expect(binding.Credentials.Password).NotTo(BeEmpty())
	Expect(binding.AppID).NotTo(BeEmpty())

	return binding
}

func (s *ServiceHelper) Unbind(bindingID string) {
	resp := DoRequest(s.BrokerAPI, NewRequest(
		"DELETE",
		fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", s.InstanceID, bindingID),
		nil,
		s.Cfg.Username,
		s.Cfg.Password,
		UriParam{Key: "service_id", Value: s.ServiceID},
		UriParam{Key: "plan_id", Value: s.PlanID},
	))
	Expect(resp.Code).To(Equal(200))
	// Response will be an empty JSON object for future compatibility
	var data map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	Expect(err).ToNot(HaveOccurred())
}

func (s *ServiceHelper) Deprovision() {
	resp := DoRequest(s.BrokerAPI, NewRequest(
		"DELETE",
		"/v2/service_instances/"+s.InstanceID,
		nil,
		s.Cfg.Username,
		s.Cfg.Password,
		UriParam{Key: "service_id", Value: s.ServiceID},
		UriParam{Key: "plan_id", Value: s.PlanID},
		UriParam{Key: "accepts_incomplete", Value: "true"},
	))
	Expect(resp.Code).To(BeEquivalentTo(202))

	var deprovisionResp brokerapi.DeprovisionResponse
	err := json.NewDecoder(resp.Body).Decode(&deprovisionResp)
	Expect(err).ToNot(HaveOccurred())

	operationState := PollForOperationCompletion(
		s.Cfg, s.BrokerAPI, s.InstanceID,
		s.ServiceID, s.PlanID, deprovisionResp.OperationData,
	)
	Expect(operationState).To(Equal("succeeded"), "returns success")
}

func (s *ServiceHelper) Provision() {
	resp := DoRequest(s.BrokerAPI, NewRequest(
		"PUT",
		"/v2/service_instances/"+s.InstanceID,
		strings.NewReader(fmt.Sprintf(`{
			"service_id": "%s",
			"plan_id": "%s",
			"organization_guid": "test-organization-id",
			"space_guid": "space-id",
			"parameters": "{}"
		}`, s.ServiceID, s.PlanID)),
		s.Cfg.Username,
		s.Cfg.Password,
		UriParam{Key: "accepts_incomplete", Value: "true"},
	))
	Expect(resp.Code).To(Equal(202))

	var provisionResp brokerapi.ProvisioningResponse
	err := json.NewDecoder(resp.Body).Decode(&provisionResp)
	Expect(err).NotTo(HaveOccurred())

	operationState := PollForOperationCompletion(
		s.Cfg, s.BrokerAPI,
		s.InstanceID, s.ServiceID, s.PlanID, provisionResp.OperationData,
	)
	Expect(operationState).To(Equal("succeeded"), "and returns success")

	// ensure deployment is in expected cluster
	deploymentName := fmt.Sprintf("%s-%s", s.Cfg.DBPrefix, s.InstanceID)
	deployment, errs := s.ComposeClient.GetDeploymentByName(deploymentName)
	Expect(errs).To(BeNil())
	clusterURL, err := url.Parse(deployment.Links.ClusterLink.HREF)
	Expect(err).ToNot(HaveOccurred())
	splitPath := strings.Split(strings.TrimRight(clusterURL.Path, "{"), "/")
	clusterID := splitPath[len(splitPath)-1]
	expectedCluster, errs := s.ComposeClient.GetClusterByName(s.Cfg.ClusterName)
	Expect(errs).To(BeNil())
	Expect(clusterID).To(Equal(expectedCluster.ID))
	Expect(expectedCluster.Type).To(Equal("private"))
}

func NewService(serviceID string, planID string) (s *ServiceHelper) {
	s = &ServiceHelper{
		InstanceID: NewUUID(),
		ServiceID:  serviceID,
		PlanID:     planID,
		Cfg: &config.Config{
			Username: randString(10),
			Password: randString(10),
			DBPrefix: "test-suite",
			APIToken: os.Getenv("COMPOSE_API_KEY"),
		},
	}
	Expect(s.Cfg.APIToken).NotTo(BeEmpty(), "Please export $COMPOSE_API_KEY")
	var err error
	s.Catalog, err = catalog.Load(strings.NewReader(`{
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
		},{
			"id": "6e9202f2-c2e1-4de8-8d4a-a8c898fc2d8c",
			"name": "elastic_search",
			"bindable": true,
			"description": "Compose Elasticsearch instance",
			"requires": [],
			"tags": [
			"elasticsearch",
			"compose"
			],
			"metadata": {
				"displayName": "Elasticsearch",
				"imageUrl": "https://static-www.elastic.co/assets/blt9a26f88bfbd20eb5/icon-elasticsearch-bb.svg",
				"longDescription": "Compose Elasticsearch instance",
				"providerDisplayName": "GOV.UK PaaS",
				"documentationUrl": "https://compose.com/databases/elasticsearch",
				"supportUrl": "https://www.cloud.service.gov.uk/support.html"
			},
			"plans": [{
				"id": "6d051078-0913-403c-9763-1d03ecee50d9",
				"name": "tiny",
				"description": "2GB Storage / 2048MB RAM.",
				"metadata": {
					"displayName": "Elasticsearch Tiny",
					"bullets": [],
					"units": 1
				}
			}]
		}]
	}`))
	Expect(err).ToNot(HaveOccurred())

	s.ComposeClient, err = compose.NewClient(s.Cfg.APIToken)
	Expect(err).NotTo(HaveOccurred())

	// select the target cluster
	// currently just the first cluster returned
	clusters, errs := s.ComposeClient.GetClusters()
	Expect(errs).To(BeNil())
	Expect(len(*clusters)).To(Equal(1))
	s.Cfg.ClusterName = (*clusters)[0].Name

	logger := lager.NewLogger("compose-broker")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, s.Cfg.LogLevel))
	s.BrokerInstance, err = broker.New(s.ComposeClient, s.Cfg, s.Catalog, logger)
	Expect(err).NotTo(HaveOccurred())

	s.BrokerAPI = brokerapi.New(s.BrokerInstance, logger, brokerapi.BrokerCredentials{
		Username: s.Cfg.Username,
		Password: s.Cfg.Password,
	})
	Expect(s.BrokerAPI).NotTo(BeNil())
	return s
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
