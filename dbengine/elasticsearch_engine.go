package dbengine

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

// FIXME: this resolves an issue with the hostname returned by Compose
// Compose may return: `cluster-name-c002.compose.direct`,
// when it should return: `cluster-name-c00.2.compose.direct`.
// A support ticket has been raised. This can be removed once the upstream support ticket is resolved.
func fixComposeDirectHostname(hostname string) string {
	re := regexp.MustCompile(`(.+-[a-z]{1}[0-9]{2})\.?(\d+)([^:]+):(\d+)`)
	if strings.Contains(hostname, "compose.direct") {
		faulty := re.FindStringSubmatch(hostname)
		hostname = fmt.Sprintf("%s.%s%s:%s", faulty[1], faulty[2], faulty[3], faulty[4])
	}
	return hostname
}

func stringToURL(input string) (*url.URL, error) {
	if input == "" {
		return nil, fmt.Errorf("Invalid empty connection String")
	}
	u, err := url.Parse(input)
	if err != nil {
		return nil, err
	}
	u.RawQuery = ""
	u.Host = fixComposeDirectHostname(u.Host)
	return u, nil
}

func parseURL(connectionStrings []string) (*ElasticSearchCredentials, error) {
	var hosts []string
	var fixedConnectionStrings []string
	var username, password string

	for i, s := range connectionStrings {
		u, err := stringToURL(s)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			username = u.User.Username()
			password, _ = u.User.Password()
		}
		hosts = append(hosts, u.Host)
		fixedConnectionStrings = append(fixedConnectionStrings, u.String())
	}

	info := ElasticSearchCredentials{
		Hosts:    hosts,
		Username: username,
		Password: password,
		URI:      fixedConnectionStrings[0],
		URIs:     fixedConnectionStrings,
	}
	return &info, nil
}

type ElasticSearchCredentials struct {
	Hosts               []string `json:"hosts"`
	Name                string   `json:"name"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	URI                 string   `json:"uri"`
	URIs                []string `json:"uris"`
	CACertificateBase64 string   `json:"ca_certificate_base64"`
}

type ElasticSearchEngine struct {
	deployment *composeapi.Deployment
}

func NewElasticSearchEngine(deployment *composeapi.Deployment) *ElasticSearchEngine {
	return &ElasticSearchEngine{deployment}
}

func (e *ElasticSearchEngine) GenerateCredentials(instanceID, bindingID string) (interface{}, error) {
	if e.deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	} else if len(e.deployment.Connection.Direct) < 1 {
		return nil, fmt.Errorf("failed to get connection string")
	}

	credentials, err := parseURL(e.deployment.Connection.Direct)
	if err != nil {
		return nil, err
	} else if credentials.Username == "" {
		return nil, fmt.Errorf("connection string did not contain a user")
	}
	credentials.CACertificateBase64 = e.deployment.CACertificateBase64

	return credentials, nil
}

func (e *ElasticSearchEngine) RevokeCredentials(instanceID, bindingID string) error {
	return nil
}
