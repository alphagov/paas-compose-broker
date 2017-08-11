package dbengine

import composeapi "github.com/compose/gocomposeapi"

type Credentials struct {
	Host                string `json:"host"`
	Port                string `json:"port"`
	Name                string `json:"name"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	URI                 string `json:"uri"`
	CACertificateBase64 string `json:"ca_certificate_base64"`
}

type DBEngine interface {
	ParseConnectionString(deployment *composeapi.Deployment) (*Credentials, error)
	CreateUser(instanceID, bindingID string, deployment *composeapi.Deployment) (*Credentials, error)
	DropUser(instanceID, bindingID string, deployment *composeapi.Deployment) error
	Open(*Credentials) error
	Close()
}
