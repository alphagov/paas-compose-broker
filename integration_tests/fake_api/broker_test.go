package fake_api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/compose/fakes"
	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
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
			Name:       fmt.Sprintf("%s-%s", dbprefix, "3d4548fe-ea5e-4ad8-bd2d-a677c76b8275"),
			Connection: composeapi.ConnectionStrings{Direct: []string{"mongodb://admin:password@aws-eu-west-1-portal.2.dblayer.com:18899,aws-eu-west-1-portal.7.dblayer.com:18899/admin?ssl=true"}},
		}

		fakeComposeClient, err = fakes.NewFakeClient()
		Expect(err).NotTo(HaveOccurred())
		fakeComposeClient.Deployments = &[]composeapi.Deployment{
			{
				ID:   "1",
				Name: fmt.Sprintf("%s-%s", dbprefix, "bd7a89b4-2a99-4ea2-a369-2ec42230af72"),
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
		req := helper.NewRequest("GET", "/v2/catalog", nil, username, password)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		body := helper.ReadResponseBody(responseRecorder.Body)
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
			param := helper.UriParam{Key: "accepts_incomplete", Value: strconv.FormatBool(acceptsIncomplete)}
			req := helper.NewRequest("PUT", path, bytes.NewBuffer(provisionDetailsJson), username, password, param)
			brokerAPI.ServeHTTP(responseRecorder, req)

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", dbprefix, instanceID),
				AccountID:    "",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "mongodb",
				Units:        1,
				SSL:          true,
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
			Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))
		})

		It("ignores user provided parameters", func() {
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
			param := helper.UriParam{Key: "accepts_incomplete", Value: strconv.FormatBool(acceptsIncomplete)}
			req := helper.NewRequest("PUT", path, bytes.NewBuffer(provisionDetailsJson), username, password, param)
			brokerAPI.ServeHTTP(responseRecorder, req)

			expectedDeploymentParams := composeapi.DeploymentParams{
				Name:         fmt.Sprintf("%s-%s", dbprefix, instanceID),
				AccountID:    "",
				Datacenter:   broker.ComposeDatacenter,
				DatabaseType: "mongodb",
				Units:        1,
				SSL:          true,
				WiredTiger:   false,
				Version:      "",
			}
			Expect(fakeComposeClient.CreateDeploymentParams).To(Equal(expectedDeploymentParams))
			Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"provision-recipe-id\",\"type\":\"provision\"}`))
		})
	})

	It("deprovisions the correct service instance", func() {
		instanceID := strings.TrimPrefix(deployment.Name, dbprefix+"-")
		path := "/v2/service_instances/" + instanceID
		req := helper.NewRequest(
			"DELETE",
			path,
			nil,
			username,
			password,
			helper.UriParam{Key: "service_id", Value: serviceID},
			helper.UriParam{Key: "plan_id", Value: planID},
			helper.UriParam{Key: "accepts_incomplete", Value: strconv.FormatBool(acceptsIncomplete)},
		)
		brokerAPI.ServeHTTP(responseRecorder, req)

		Expect(fakeComposeClient.DeprovisionDeploymentID).To(Equal(deployment.ID))
		Expect(responseRecorder.Code).To(Equal(http.StatusAccepted))
		body := helper.ReadResponseBody(responseRecorder.Body)
		Expect(string(body)).To(ContainSubstring(`{\"recipe_id\":\"deprovision-recipe-id\",\"type\":\"deprovision\"}`))
	})

	It("can provide a service instance binding", func() {
		instanceID := strings.TrimPrefix(deployment.Name, dbprefix+"-")
		path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
		bindingDetailsJson := []byte(fmt.Sprintf(`{
				"service_id": "%s",
				"plan_id": "%s",
				"bind_resource": {
					"app_guid": "%s"
				},
				"parameters": "%s"
			}`, serviceID, planID, appGuid, paramJSON))
		req := helper.NewRequest(
			"PUT",
			path,
			bytes.NewBuffer(bindingDetailsJson),
			username,
			password,
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
		req := helper.NewRequest(
			"DELETE",
			path,
			bytes.NewBuffer(unbindingDetailsJson),
			username,
			password,
		)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusOK))
	})

	It("will not let you update the plan", func() {
		instanceID := strings.TrimPrefix(deployment.Name, dbprefix+"-")
		path := fmt.Sprintf("/v2/service_instances/%s", instanceID)
		newPlanID := "Plan-2"
		provisionDetailsJson := []byte(fmt.Sprintf(`{
				"service_id": "%s",
				"plan_id": "%s",
				"previous_values": {
					"plan_id": "%s"
				},
				"parameters": %s
			}`, serviceID, newPlanID, planID, paramJSON))
		req := helper.NewRequest(
			"PATCH",
			path,
			bytes.NewBuffer(provisionDetailsJson),
			username,
			password,
			helper.UriParam{Key: "accepts_incomplete", Value: strconv.FormatBool(acceptsIncomplete)},
		)
		brokerAPI.ServeHTTP(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := helper.ReadResponseBody(responseRecorder.Body)
		Expect(string(body)).To(ContainSubstring("changing plans is not currently supported"))
	})

	Context("when checking the status of the last operation", func() {
		var (
			path string
			req  *http.Request
		)

		BeforeEach(func() {
			path = fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
			req = helper.NewRequest(
				"GET",
				path,
				nil,
				username,
				password,
				helper.UriParam{Key: "service_id", Value: serviceID},
				helper.UriParam{Key: "plan_id", Value: planID},
				helper.UriParam{Key: "operation", Value: "{\"recipe_id\":\"recipe-id\",\"type\":\"provision\"}"},
			)
		})

		It("returns an error when unable to get the recipe", func() {
			fakeComposeClient.GetRecipeErr = fmt.Errorf("error: failed to get recipe by ID")
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring(`{"description":"error: failed to get recipe by ID"}`))
		})

		It("returns a failed state when the Compose recipe status is not recognised", func() {
			fakeComposeClient.GetRecipeStatus = "some-unknown-recipe-status"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("failed"))
		})

		It("returns OK when last operation has completed", func() {
			fakeComposeClient.GetRecipeStatus = "complete"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("succeeded"))
		})

		It("returns OK when last operation is still running", func() {
			fakeComposeClient.GetRecipeStatus = "running"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("in progress"))
		})

		It("returns OK when last operation is waiting to run", func() {
			fakeComposeClient.GetRecipeStatus = "waiting"
			brokerAPI.ServeHTTP(responseRecorder, req)
			Expect(fakeComposeClient.GetRecipeID).To(Equal("recipe-id"))
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			body := helper.ReadResponseBody(responseRecorder.Body)
			Expect(string(body)).To(ContainSubstring("in progress"))
		})
	})
})
