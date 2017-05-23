package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

type expectedCatalog struct {
	Services []Service `json:"services"`
}

type Service struct {
	Name string `json:"name"`
}

var (
	responseRecorder  *httptest.ResponseRecorder
	instanceID        string
	serviceID         string
	planID            string
	bindingID         string
	appGuid           string
	paramJSON         string
	acceptsIncomplete bool
)

var _ = Describe("Broker", func() {

	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()
		instanceID = uuid.NewV4().String()
		serviceID = "Service-1"
		planID = "Plan-1"
		bindingID = uuid.NewV4().String()
		appGuid = uuid.NewV4().String()
		paramJSON = "{}"
		acceptsIncomplete = true
	})

	It("serves the catalog endpoint", func() {
		req := newRequest("GET", "/v2/catalog", nil)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusOK))
		body := readResponseBody(resp)
		var returnedCatalog expectedCatalog
		err := json.Unmarshal(body, &returnedCatalog)
		Expect(err).ToNot(HaveOccurred())
		Expect(returnedCatalog.Services[0].Name).To(Equal("mongo"))
	})

	It("serves the provision endpoint", func() {
		path := "/v2/service_instances/" + instanceID
		provisionDetailsJson := []byte(fmt.Sprintf(`
			{
				"service_id": "%s",
				"plan_id": "%s",
				"organization_guid": "test-organization-id",
				"space_guid": "space-id",
				"parameters": %s
			}
			`, serviceID, planID, paramJSON))
		param := uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)}
		req := newRequest("PUT", path, bytes.NewBuffer(provisionDetailsJson), param)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(resp)
		Expect(string(body)).To(ContainSubstring("Can't provision an instance"))
	})

	It("serves the deprovision endpoint", func() {
		path := "/v2/service_instances/" + instanceID
		req := newRequest(
			"DELETE",
			path,
			nil,
			uriParam{key: "service_id", value: serviceID},
			uriParam{key: "plan_id", value: planID},
			uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)},
		)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(resp)
		Expect(string(body)).To(ContainSubstring("Can't deprovision an instance"))
	})

	It("serves the bind endpoint", func() {
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
		req := newRequest(
			"PUT",
			path,
			bytes.NewBuffer(bindingDetailsJson),
		)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(resp)
		Expect(string(body)).To(ContainSubstring("Can't bind an instance"))
	})

	It("serves the unbind endpoint", func() {
		path := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
		unbindingDetailsJson := []byte(fmt.Sprintf(`
			{
				"service_id": "%s",
				"plan_id": "%s",
			}`,
			serviceID,
			planID,
		))
		req := newRequest(
			"DELETE",
			path,
			bytes.NewBuffer(unbindingDetailsJson),
		)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(resp)
		Expect(string(body)).To(ContainSubstring("Can't unbind an instance"))
	})

	It("serves the update endpoint", func() {
		path := fmt.Sprintf("/v2/service_instances/%s", instanceID)
		newPlanID := "Plan-2"
		provisionDetailsJson := []byte(fmt.Sprintf(`
			{
				"service_id": "%s",
				"plan_id": "%s",
				"previous_values": {
					"plan_id": "%s"
				},
				"parameters": %s
			}
		`, serviceID, planID, newPlanID, paramJSON))
		req := newRequest(
			"PATCH",
			path,
			bytes.NewBuffer(provisionDetailsJson),
			uriParam{key: "accepts_incomplete", value: strconv.FormatBool(acceptsIncomplete)},
		)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(resp)
		Expect(string(body)).To(ContainSubstring("Can't update an instance"))
	})

	It("serves the last operation endpoint", func() {
		path := fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID)
		req := newRequest(
			"GET",
			path,
			nil,
			uriParam{key: "service_id", value: serviceID},
			uriParam{key: "plan_id", value: planID},
			uriParam{key: "operation", value: "some ID returned by an async operation"},
		)
		resp := serveRequest(responseRecorder, req)
		Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
		body := readResponseBody(resp)
		Expect(string(body)).To(ContainSubstring("Can't check last operation"))
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

func serveRequest(w *httptest.ResponseRecorder, req *http.Request) *http.Response {
	brokerAPI.ServeHTTP(w, req)
	return w.Result()
}

func readResponseBody(resp *http.Response) []byte {
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return body
}
