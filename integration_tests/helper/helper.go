package helper

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose"
	"github.com/alphagov/paas-compose-broker/config"
	"github.com/alphagov/paas-compose-broker/dbengine"

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
	Catalog        *catalog.Catalog
	Cfg            *config.Config
	Logger         lager.Logger
	BrokerInstance *broker.Broker
	BrokerAPI      http.Handler
	ComposeClient  compose.Client
	Provider       dbengine.Provider
}

func (s *ServiceHelper) Bind(appID string) (binding *BindingData) {
	binding = &BindingData{
		ID:    NewUUID(),
		AppID: appID,
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
		UriParam{Key: "accepts_incomplete", Value: "true"},
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
		Provider: dbengine.NewProviderService(),
	}
	Expect(s.Cfg.APIToken).NotTo(BeEmpty(), "Please export $COMPOSE_API_KEY")
	b, err := ioutil.ReadFile("./../examples/catalog.json")
	Expect(err).NotTo(HaveOccurred())
	s.Catalog, err = catalog.Load(bytes.NewReader(b))
	Expect(err).ToNot(HaveOccurred())
	Expect(len(s.Catalog.Services)).To(BeNumerically(">", 0))

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

	s.BrokerInstance, err = broker.New(s.ComposeClient, s.Provider, s.Cfg, s.Catalog, logger)
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

func MongoConnection(uri, caBase64 string) (*mgo.Session, error) {
	mongoUrl, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	password, _ := mongoUrl.User.Password()
	return mgo.DialWithInfo(&mgo.DialInfo{
		Addrs: strings.Split(mongoUrl.Host, ","),
		//Addrs:    []string{strings.Split(mongoUrl.Host, ",")[0] + ":" + mongoUrl.Port()},
		Database: strings.TrimPrefix(mongoUrl.Path, "/"),
		Timeout:  10 * time.Second,
		Username: mongoUrl.User.Username(),
		Password: password,
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			ca, err := base64.StdEncoding.DecodeString(caBase64)
			if err != nil {
				return nil, err
			}
			roots := x509.NewCertPool()
			roots.AppendCertsFromPEM(ca)
			return tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr.String(), &tls.Config{
				RootCAs: roots,
			})
		},
	})
}
