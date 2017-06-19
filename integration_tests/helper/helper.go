package helper

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/gomega"
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
