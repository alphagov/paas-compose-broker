package dbengine

import (
	"fmt"
	"net/url"

	composeapi "github.com/compose/gocomposeapi"
)

type RedisCredentials struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	URI      string `json:"uri"`
}

type RedisEngine struct {
	deployment *composeapi.Deployment
}

func NewRedisEngine(deployment *composeapi.Deployment) *RedisEngine {
	return &RedisEngine{deployment}
}

func (e *RedisEngine) GenerateCredentials(instanceID, bindingID string) (interface{}, error) {
	if e.deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	} else if len(e.deployment.Connection.Direct) < 1 {
		return nil, fmt.Errorf("failed to get connection string")
	}

	u, err := url.Parse(e.deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	}

	password, _ := u.User.Password()
	return &RedisCredentials{
		Host:     u.Hostname(),
		Port:     u.Port(),
		Username: u.User.Username(),
		Password: password,
		URI:      u.String(),
	}, nil
}

func (e *RedisEngine) RevokeCredentials(instanceID, bindingID string) error {
	return nil
}
