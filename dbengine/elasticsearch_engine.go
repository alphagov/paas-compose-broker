package dbengine

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

type ElasticSearchCredentials struct {
	Host                string `json:"host"`
	Port                string `json:"port"`
	Name                string `json:"name"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	URI                 string `json:"uri"`
	CACertificateBase64 string `json:"ca_certificate_base64"`
	AuthSource          string `json:"auth_source"`
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

	u, err := url.Parse(e.deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	} else if u.User == nil {
		return nil, fmt.Errorf("connection string did not contain a user")
	}

	// FIXME: Follow up story should fix connection string handling.
	// Right now we are hardcoding first host from the comma delimited list that Compose provides.
	// url.Parse() parses connection string wrongly and doesn't return an error
	// so url.Port() returns port like "18899,aws-eu-west-1-portal.7.dblayer.com:18899"
	port := strings.Split(u.Port(), ",")[0]
	u.Host = fmt.Sprintf("%s:%s", u.Hostname(), port)

	// FIXME: this resolves an issue with the hostname returned by Compose
	// Compose may return: `cluster-name-c002.compose.direct`,
	// when it should return: `cluster-name-c00.2.compose.direct`.
	// A support ticket has been raised. This can be removed once the upstream support ticket is resolved.
	if strings.Contains(u.Hostname(), "compose.direct") {
		re := regexp.MustCompile(`(.+-[a-z]{1}[0-9]{2})\.?(\d+)(.+)`)
		faulty := re.FindStringSubmatch(u.Hostname())
		u.Host = fmt.Sprintf("%s.%s%s:%s", faulty[1], faulty[2], faulty[3], port)
	}

	password, _ := u.User.Password()
	return &ElasticSearchCredentials{
		Host:                u.Hostname(),
		Port:                u.Port(),
		URI:                 u.String(),
		Username:            u.User.Username(),
		Password:            password,
		CACertificateBase64: e.deployment.CACertificateBase64,
	}, nil
}

func (e *ElasticSearchEngine) RevokeCredentials(instanceID, bindingID string) error {
	return nil
}
