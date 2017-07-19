package helper

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"

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

func MongoConnection(uri, caBase64 string) (*mgo.Session, error) {
	// This is work around for https://github.com/go-mgo/mgo/issues/84
	uri = strings.TrimSuffix(uri, "?ssl=true")
	mongourl, err := mgo.ParseURL(uri)
	if err != nil {
		return nil, err
	}

	// Compose has self-signed certs for mongo. Make sure we verify it against CA certificate brovided in binding
	ca, err := base64.StdEncoding.DecodeString(caBase64)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(ca)

	tlsConfig := &tls.Config{RootCAs: roots}
	Expect(tlsConfig.InsecureSkipVerify).To(BeFalse())

	mongourl.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
		return tls.Dial("tcp", addr.String(), tlsConfig)
	}
	mongourl.Timeout = 10 * time.Second
	session, err := mgo.DialWithInfo(mongourl)

	return session, err
}
