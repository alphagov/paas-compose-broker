package dbengine

import (
	"fmt"
	"net/url"

	composeapi "github.com/compose/gocomposeapi"
)

type RedisEngine struct {
	credentials *Credentials
}

func NewRedisEngine() *RedisEngine {
	return &RedisEngine{}
}

func (e *RedisEngine) CreateUser(instanceID, bindingID string, deployment *composeapi.Deployment) (*Credentials, error) {
	return e.credentials, nil
}

func (e *RedisEngine) DropUser(instanceID, bindingID string, deployment *composeapi.Deployment) error {
	return nil
}

func (e *RedisEngine) Open(creds *Credentials) error {
	e.credentials = creds
	return nil
}

func (e *RedisEngine) Close() {}

func (e *RedisEngine) ParseConnectionString(deployment *composeapi.Deployment) (*Credentials, error) {
	if deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	}

	u, err := url.Parse(deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	}

	password, _ := u.User.Password()
	return &Credentials{
		Host:     u.Hostname(),
		Port:     u.Port(),
		URI:      u.String(),
		Username: u.User.Username(),
		Password: password,
		Name:     "",
	}, nil
}
