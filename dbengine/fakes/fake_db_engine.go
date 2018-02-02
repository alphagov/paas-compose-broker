package fakes

import (
	composeapi "github.com/compose/gocomposeapi"
)

type FakeCredentials struct {
	Host                string `json:"host"`
	Port                string `json:"port"`
	Name                string `json:"name"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	URI                 string `json:"uri"`
	CACertificateBase64 string `json:"ca_certificate_base64"`
}

type FakeDBEngine struct {
	deployment *composeapi.Deployment
}

func (e *FakeDBEngine) GenerateCredentials(instanceID, bindingID string) (interface{}, error) {
	return &FakeCredentials{
		Host:                "localhost",
		Port:                "27017",
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
