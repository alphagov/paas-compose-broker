package dbengine

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/alphagov/paas-compose-broker/client/elastic"
	composeapi "github.com/compose/gocomposeapi"
)

type ElasticSearchEngine struct {
	credentials *Credentials
}

func NewElasticSearchEngine() *ElasticSearchEngine {
	return &ElasticSearchEngine{}
}

func (e *ElasticSearchEngine) CreateUser(instanceID, bindingID string, deployment *composeapi.Deployment) (*Credentials, error) {
	return e.credentials, nil
}

func (e *ElasticSearchEngine) DropUser(instanceID, bindingID string, deployment *composeapi.Deployment) error {
	return nil
}

func (e *ElasticSearchEngine) Open(creds *Credentials) error {
	if creds == nil {
		return fmt.Errorf("credentials: not provided")
	}

	e.credentials = creds

	err := e.testConnection()
	if err != nil {
		return fmt.Errorf("connection refused: %s", err)
	}

	return nil
}

func (e *ElasticSearchEngine) Close() {}

func (e *ElasticSearchEngine) ParseConnectionString(deployment *composeapi.Deployment) (*Credentials, error) {
	if deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	}

	u, err := url.Parse(deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	}

	// FIXME: this resolves an issue with the hostname returned by Compose
	// Compose may return: `cluster-name-c002.compose.direct`,
	// when it should return: `cluster-name-c00.2.compose.direct`.
	// A support ticket has been raised. This can be removed once the upstream support ticket is resolved.
	if strings.Contains(u.Hostname(), "compose.direct") {
		re := regexp.MustCompile(`(.+-[a-z]{1}[0-9]{2})\.?(\d+)(.+)`)
		faulty := re.FindStringSubmatch(u.Hostname())
		u.Host = fmt.Sprintf("%s.%s%s:%s", faulty[1], faulty[2], faulty[3], u.Port())
	}

	password, _ := u.User.Password()
	return &Credentials{
		Host:                u.Hostname(),
		Port:                u.Port(),
		URI:                 u.String(),
		Username:            u.User.Username(),
		Password:            password,
		Name:                "",
		CACertificateBase64: deployment.CACertificateBase64,
	}, nil
}

func (e *ElasticSearchEngine) testConnection() error {
	httpClient, err := SetupHTTPClient(e.credentials.CACertificateBase64)
	if err != nil {
		return err
	}

	client, err := elastic.New(e.credentials.URI, httpClient)
	if err != nil {
		return err
	}

	_, err = client.Version()

	if err != nil {
		return err
	}

	return nil
}
