package dbengine

type DBEngine interface {
	GenerateCredentials(instanceID, bindingID string) (interface{}, error)
	RevokeCredentials(instanceID, bindingID string) error
}
