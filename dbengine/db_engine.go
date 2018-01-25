package dbengine

type Credentials struct {
	Hosts               []string `json:"hosts"`
	Name                string   `json:"name"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	URI                 string   `json:"uri"`
	CACertificateBase64 string   `json:"ca_certificate_base64"`
	AuthSource          string   `json:"auth_source,omitempty"`
}

type DBEngine interface {
	GenerateCredentials(instanceID, bindingID string) (*Credentials, error)
	RevokeCredentials(instanceID, bindingID string) error
}
