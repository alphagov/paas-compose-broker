package real_api_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"
)

var (
	brokerAPI http.Handler
)

const (
	serviceID = "36f8bf47-c9e7-46d9-880f-5dfc838d05cb"
	planID    = "fdfd4fc1-ce69-451c-a436-c2e2795b9abe"
	timeout   = 10 * time.Second
)

type bindingResponse struct {
	Credentials map[string]string `json:"credentials"`
}

type Person struct {
	Name  string
	Phone string
}

var _ = Describe("Broker with real Compose client", func() {

	var (
		composeClient     *composeapi.Client
		serviceBroker     *broker.Broker
		err               error
		credentials       brokerapi.BrokerCredentials
		responseRecorder  *httptest.ResponseRecorder
		instanceID        string
		bindingID         string
		appGuid           string
		paramJSON         string
		acceptsIncomplete bool
		data              bindingResponse
	)

	BeforeEach(func() {

		composeClient, err = composeapi.NewClient(newConfig.APIToken)
		Expect(err).NotTo(HaveOccurred())
		serviceBroker, err = broker.New(composeClient, newConfig, &newCatalog, logger)
		Expect(err).NotTo(HaveOccurred())
		credentials = brokerapi.BrokerCredentials{
			Username: newConfig.Username,
			Password: newConfig.Password,
		}
		brokerAPI = brokerapi.New(serviceBroker, logger, credentials)

		responseRecorder = httptest.NewRecorder()
		instanceID = uuid.NewV4().String()
		bindingID = uuid.NewV4().String()
		appGuid = uuid.NewV4().String()
		paramJSON = "{}"
		acceptsIncomplete = true
	})

	AfterEach(func() {
		By("De-provisioning an instance", func() {
			path := "/v2/service_instances/" + instanceID
			request := helper.NewRequest(
				"DELETE",
				path,
				nil,
				username,
				password,
				helper.UriParam{Key: "service_id", Value: serviceID},
				helper.UriParam{Key: "plan_id", Value: planID},
				helper.UriParam{Key: "accepts_incomplete", Value: strconv.FormatBool(acceptsIncomplete)},
			)
			brokerAPI.ServeHTTP(responseRecorder, request)
			Expect(responseRecorder.Code).To(BeEquivalentTo(http.StatusAccepted))
			body := helper.ReadResponseBody(responseRecorder.Body)
			var deprovisionResp brokerapi.DeprovisionResponse
			err = json.Unmarshal(body, &deprovisionResp)
			Expect(err).ToNot(HaveOccurred())
			operationState := pollForOperationCompletion(instanceID, serviceID, planID, deprovisionResp.OperationData)
			Expect(operationState).To(Equal("succeeded"), "returns success")
		})
	})

	It("uses Compose API", func() {
		By("provisioning an instance", func() {
			path := "/v2/service_instances/" + instanceID
			mongoVersion := "3.2.11"
			provisionDetailsJson := []byte(fmt.Sprintf(`
				{
					"service_id": "%s",
					"plan_id": "%s",
					"organization_guid": "test-organization-id",
					"space_guid": "space-id",
					"parameters": {
						"disable_ssl": true,
						"wired_tiger": true,
						"version": "%s"
					}
				}
			`, serviceID, planID, mongoVersion))
			param := helper.UriParam{Key: "accepts_incomplete", Value: strconv.FormatBool(acceptsIncomplete)}
			request := helper.NewRequest("PUT", path, bytes.NewBuffer(provisionDetailsJson), username, password, param)
			brokerAPI.ServeHTTP(responseRecorder, request)
			Expect(responseRecorder.Code).To(BeEquivalentTo(http.StatusAccepted))
			body := helper.ReadResponseBody(responseRecorder.Body)
			var provisionResp brokerapi.ProvisioningResponse
			err = json.Unmarshal(body, &provisionResp)
			Expect(err).ToNot(HaveOccurred())
			operationState := pollForOperationCompletion(instanceID, serviceID, planID, provisionResp.OperationData)
			Expect(operationState).To(Equal("succeeded"), "and returns success")
		})
		By("binding an instance", func() {
			path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
			bindingDetailsJson := []byte(fmt.Sprintf(`
				{
					"service_id": "%s",
					"plan_id": "%s",
					"bind_resource": {
						"app_guid": "%s"
					},
					"parameters": "%s"
				}`,
				serviceID,
				planID,
				appGuid,
				paramJSON,
			))
			req := helper.NewRequest(
				"PUT",
				path,
				bytes.NewBuffer(bindingDetailsJson),
				username,
				password,
			)

			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))

			body := helper.ReadResponseBody(responseRecorder.Body)
			err := json.NewDecoder(bytes.NewReader(body)).Decode(&data)
			Expect(err).ToNot(HaveOccurred())
		})

		By("connecting to an instance", func() {
			// This is work around for https://github.com/go-mgo/mgo/issues/84
			uri := strings.TrimSuffix(data.Credentials["uri"], "?ssl=true")
			mongourl, err := mgo.ParseURL(uri)
			Expect(err).ToNot(HaveOccurred())

			// Compose has self-signed certs for mongo nd we dont pass ca_certificate_base64 in the binding
			tlsConfig := &tls.Config{
				InsecureSkipVerify: true,
			}
			mongourl.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
				conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
				return conn, err
			}
			mongourl.Timeout = timeout
			session, err := mgo.DialWithInfo(mongourl)
			Expect(err).ToNot(HaveOccurred())
			defer session.Close()

			name := "John Jones"
			phone := "+447777777777"
			db := session.DB("test").C("people")
			err = db.Insert(&Person{Name: name, Phone: phone})
			Expect(err).ToNot(HaveOccurred())

			result := Person{}
			err = db.Find(bson.M{"name": name}).One(&result)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Phone).To(Equal(phone))
			Expect(result.Name).To(Equal(name))
		})
	})
})

func pollForOperationCompletion(instanceID, serviceID, planID, operation string) string {
	var state string
	var err error

	fmt.Fprint(GinkgoWriter, "Polling for Instance Operation to complete")
	Eventually(
		func() string {
			fmt.Fprint(GinkgoWriter, ".")
			path := fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
			request := helper.NewRequest(
				"GET",
				path,
				nil,
				username,
				password,
				helper.UriParam{Key: "service_id", Value: serviceID},
				helper.UriParam{Key: "plan_id", Value: planID},
				helper.UriParam{Key: "operation", Value: operation},
			)

			responseRecorder := httptest.NewRecorder()
			brokerAPI.ServeHTTP(responseRecorder, request)
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := helper.ReadResponseBody(responseRecorder.Body)
			var lastOperation brokerapi.LastOperationResponse
			err = json.Unmarshal(body, &lastOperation)
			Expect(err).ToNot(HaveOccurred())
			state = string(lastOperation.State)
			return state
		},
		INSTANCE_CREATE_TIMEOUT,
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
