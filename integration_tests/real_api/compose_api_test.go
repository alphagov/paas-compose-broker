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

			var data struct {
				Credentials struct {
					Host     string `json:"host"`
					Port     string `json:"port"`
					Name     string `json:"name"`
					Username string `json:"username"`
					Password string `json:"password"`
					URI      string `json:"uri"`
					JDBCURI  string `json:"jdbcuri"`
				} `json:"credentials"`
			}
			body := helper.ReadResponseBody(responseRecorder.Body)
			err := json.NewDecoder(bytes.NewReader(body)).Decode(&data)
			Expect(err).ToNot(HaveOccurred())
			Expect(data.Credentials.Host).ToNot(BeEmpty())
			Expect(data.Credentials.Port).ToNot(BeEmpty())
			Expect(data.Credentials.Name).ToNot(BeEmpty())
			Expect(data.Credentials.Username).ToNot(BeEmpty())
			Expect(data.Credentials.Password).ToNot(BeEmpty())
			Expect(data.Credentials.URI).ToNot(BeEmpty())
			Expect(data.Credentials.JDBCURI).ToNot(BeEmpty())

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
