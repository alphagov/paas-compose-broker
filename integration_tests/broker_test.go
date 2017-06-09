package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/compose/fakes"
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"
)

type expectedCatalog struct {
	Services []Service `json:"services"`
}

type Service struct {
	Name string `json:"name"`
}

type bindingResponse struct {
	Credentials map[string]string `json:"credentials"`
}

var (
	brokerAPI http.Handler
)

const (
	serviceID = "36f8bf47-c9e7-46d9-880f-5dfc838d05cb"
	planID    = "fdfd4fc1-ce69-451c-a436-c2e2795b9abe"
)

var _ = Describe("Broker with fake Compose client", func() {

	var (
		fakeComposeClient *fakes.FakeComposeClient
		serviceBroker     *broker.Broker
		err               error
		credentials       brokerapi.BrokerCredentials
		responseRecorder  *httptest.ResponseRecorder
		deployment        composeapi.Deployment
		instanceID        string
		bindingID         string
		appGuid           string
		paramJSON         string
		acceptsIncomplete bool
	)

	BeforeEach(func() {
		deployment = composeapi.Deployment{
			ID:         "2",
			Name:       "3d4548fe-ea5e-4ad8-bd2d-a677c76b8275",
			Connection: composeapi.ConnectionStrings{Direct: []string{"mongodb://admin:password@aws-eu-west-1-portal.2.dblayer.com:18899,aws-eu-west-1-portal.7.dblayer.com:18899/admin?ssl=true"}},
		}

		fakeComposeClient, err = fakes.NewFakeClient()
		Expect(err).NotTo(HaveOccurred())
		fakeComposeClient.Deployments = &[]composeapi.Deployment{
			{
				ID:   "1",
				Name: "bd7a89b4-2a99-4ea2-a369-2ec42230af72",
			}, deployment,
		}

		serviceBroker, err = broker.New(fakeComposeClient, newConfig, &newCatalog, logger)
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

	It("serves the catalog endpoint", func() {
		req := newRequest("GET", "/v2/catalog", nil)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		body := readResponseBody(responseRecorder.Body)
		var returnedCatalog expectedCatalog
		err := json.Unmarshal(body, &returnedCatalog)
		Expect(err).ToNot(HaveOccurred())
		Expect(returnedCatalog.Services[0].Name).To(Equal("mongodb"))
	})

	Context("when provisioning", func() {
		It("supplies the correct parameters", func() {
			path := "/v2/service_instances/" + instanceID
			provisionDetailsJson := []byte(fmt.Sprintf(`{
				  "service_id": "%s",
				  "plan_id": "%s",
				  "organization_guid": "test-organization-id",
				  "space_guid": "space-id",
				  "parameters": %s
			  }`, serviceID, planID, paramJSON))
			param := uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)}
			req := newRequest("PUT", path, bytes.NewBuffer(provisionDetailsJson), param)
			brokerAPI.ServeHTTP(responseRecorder, req)

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         instanceID,
				AccountID:    os.Getenv("ACCOUNT_ID"),
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "mongodb",
				Units:        1,
				SSL:          true,
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
			Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring(`{"operation":"provision-recipe-id"}`))
		})

		It("allows user provided parameters", func() {
			path := "/v2/service_instances/" + instanceID
			provisionDetailsJson := []byte(fmt.Sprintf(`{
				  "service_id": "%s",
				  "plan_id": "%s",
				  "organization_guid": "test-organization-id",
				  "space_guid": "space-id",
				  "parameters": {
					  "disable_ssl": true,
					  "wired_tiger": true,
					  "version": "1"
				  }
			  }`, serviceID, planID))
			param := uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)}
			req := newRequest("PUT", path, bytes.NewBuffer(provisionDetailsJson), param)
			brokerAPI.ServeHTTP(responseRecorder, req)

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         instanceID,
				AccountID:    os.Getenv("ACCOUNT_ID"),
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "mongodb",
				Units:        1,
				SSL:          true,
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
			Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring(`{"operation":"provision-recipe-id"}`))
		})
	})

	It("deprovisions the correct service instance", func() {
		path := "/v2/service_instances/" + deployment.Name
		req := newRequest(
			"DELETE",
			path,
			nil,
			uriParam{key: "service_id", value: serviceID},
			uriParam{key: "plan_id", value: planID},
			uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)},
		)
		brokerAPI.ServeHTTP(responseRecorder, req)

		Expect(fakeComposeClient.DeprovisionDeploymentID).To(Equal(deployment.ID))
		Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))
		body := readResponseBody(responseRecorder.Body)
		Expect(string(body)).To(ContainSubstring(`{"operation":"deprovision-recipe-id"}`))
	})

	It("can provide a service instance binding", func() {
		path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", deployment.Name, bindingID)
		bindingDetailsJson := []byte(fmt.Sprintf(`{
				"service_id": "%s",
				"plan_id": "%s",
				"bind_resource": {
					"app_guid": "%s"
				},
				"parameters": "%s"
			}`, serviceID, planID, appGuid, paramJSON))
		req := newRequest(
			"PUT",
			path,
			bytes.NewBuffer(bindingDetailsJson),
		)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusCreated))

		var data bindingResponse

		body := helper.ReadResponseBody(responseRecorder.Body)
		err := json.NewDecoder(bytes.NewReader(body)).Decode(&data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data.Credentials["host"]).To(Equal("aws-eu-west-1-portal.2.dblayer.com"))
		Expect(data.Credentials["port"]).To(Equal("18899"))
		Expect(data.Credentials["name"]).To(Equal("/admin?ssl=true"))
		Expect(data.Credentials["username"]).To(Equal("admin"))
		Expect(data.Credentials["password"]).To(Equal("password"))
		Expect(data.Credentials["uri"]).To(Equal("mongodb://admin:password@aws-eu-west-1-portal.2.dblayer.com:18899,aws-eu-west-1-portal.7.dblayer.com:18899/admin?ssl=true"))
		Expect(data.Credentials["jdbcuri"]).To(Equal("jdbc:mongodb://aws-eu-west-1-portal.2.dblayer.com:18899//admin?ssl=true?user=admin&password=password"))
	})

	It("serves the unbind endpoint", func() {
		path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
		unbindingDetailsJson := []byte(fmt.Sprintf(`{
				"service_id": "%s",
				"plan_id": "%s",
			}`, serviceID, planID))
		req := newRequest(
			"DELETE",
			path,
			bytes.NewBuffer(unbindingDetailsJson),
		)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusOK))
	})

	It("will not let you update the plan", func() {
		path := fmt.Sprintf("/v2/service_instances/%s", deployment.Name)
		newPlanID := "Plan-2"
		provisionDetailsJson := []byte(fmt.Sprintf(`{
				"service_id": "%s",
				"plan_id": "%s",
				"previous_values": {
					"plan_id": "%s"
				},
				"parameters": %s
			}`, serviceID, newPlanID, planID, paramJSON))
		req := newRequest(
			"PATCH",
			path,
			bytes.NewBuffer(provisionDetailsJson),
			uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)},
		)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(responseRecorder.Body)
		Expect(string(body)).To(ContainSubstring("changing plans is not currently supported"))
	})

	Context("when checking the status of the last operation", func() {
		var (
			path string
			req  *http.Request
		)

		BeforeEach(func() {
			path = fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
			req = newRequest(
				"GET",
				path,
				nil,
				uriParam{key: "service_id", value: serviceID},
				uriParam{key: "plan_id", value: planID},
				uriParam{key: "operation", value: "recipe-id"},
			)
		})

		It("returns an error when unable to get the recipe", func() {
			fakeComposeClient.GetRecipeErr = fmt.Errorf("error: failed to get recipe by ID")
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring(`{"description":"error: failed to get recipe by ID"}`))
		})

		It("returns a failed state when the Compose recipe status is not recognised", func() {
			fakeComposeClient.GetRecipeStatus = "some-unknown-recipe-status"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("failed"))
		})

		It("returns OK when last operation has completed", func() {
			fakeComposeClient.GetRecipeStatus = "complete"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("succeeded"))
		})

		It("returns OK when last operation is still running", func() {
			fakeComposeClient.GetRecipeStatus = "running"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("in progress"))
		})

		It("returns OK when last operation is waiting to run", func() {
			fakeComposeClient.GetRecipeStatus = "waiting"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := readResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("in progress"))
		})
	})
})

type uriParam struct {
	key   string
	value string
}

func newRequest(method, path string, body io.Reader, params ...uriParam) *http.Request {
	brokerUrl := fmt.Sprintf("http://%s", "127.0.0.1:"+listenPort+path)
	req := httptest.NewRequest(method, brokerUrl, body)
	req.SetBasicAuth(username, password)
	q := req.URL.Query()
	for _, p := range params {
		q.Add(p.key, p.value)
	}
	req.URL.RawQuery = q.Encode()
	return req
}

func readResponseBody(responseBody *bytes.Buffer) []byte {
	body, err := ioutil.ReadAll(responseBody)
	Expect(err).ToNot(HaveOccurred())
	return body
}
