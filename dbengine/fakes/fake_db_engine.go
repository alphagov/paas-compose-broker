package fakes

import (
	"net/url"
	"strings"

	"github.com/alphagov/paas-compose-broker/dbengine"
	composeapi "github.com/compose/gocomposeapi"
)

type FakeDBEngine struct{}

func (f *FakeDBEngine) ParseConnectionString(deployment *composeapi.Deployment) (*dbengine.Credentials, error) {
	u, err := url.Parse(deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	}

	password, _ := u.User.Password()
	return &dbengine.Credentials{
		Host:                u.Hostname(),
		Port:                u.Port(),
		URI:                 u.String(),
		Username:            u.User.Username(),
		Password:            password,
		Name:                strings.Split(u.EscapedPath(), "/")[1],
		CACertificateBase64: deployment.CACertificateBase64,
	}, nil
}

func (f *FakeDBEngine) CreateUser(instanceID, bindingID string, deployment *composeapi.Deployment) (*dbengine.Credentials, error) {
	credentials, err := f.ParseConnectionString(deployment)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

func (f *FakeDBEngine) DropUser(instanceID, bindingID string, deployment *composeapi.Deployment) error {
	return nil
}

func (f *FakeDBEngine) Open(*dbengine.Credentials) error {
	return nil
}

func (f *FakeDBEngine) Close() {
	return
}
