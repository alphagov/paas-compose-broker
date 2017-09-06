package dbengine

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
	GenerateCredentials(instanceID, bindingID string) (*Credentials, error)
	RevokeCredentials(instanceID, bindingID string) error
}
