package fakes

import (
	"github.com/alphagov/paas-compose-broker/dbengine"
	composeapi "github.com/compose/gocomposeapi"
)

type FakeDBEngine struct {
	deployment *composeapi.Deployment
}

func (e *FakeDBEngine) GenerateCredentials(instanceID, bindingID string) (*dbengine.Credentials, error) {
	return &dbengine.Credentials{
		Hosts:               []string{"localhost"},
		URI:                 "fake://fadmin:fpass@fakehost.com:601601/fakedb",
		Username:            "user",
		Password:            "fpass",
		Name:                "db_1111",
		CACertificateBase64: "AAAA",
	}, nil
}

func (f *FakeDBEngine) RevokeCredentials(instanceID, bindingID string) error {
	return nil
}
