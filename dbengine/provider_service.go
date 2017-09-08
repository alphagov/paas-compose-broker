package dbengine

import (
	"fmt"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

type ProviderService struct{}

func NewProviderService() *ProviderService {
	return &ProviderService{}
}

func (p *ProviderService) GetDBEngine(deployment *composeapi.Deployment) (DBEngine, error) {
	switch strings.ToLower(deployment.Type) {
	case "mongodb":
		return NewMongoEngine(deployment), nil
	case "redis":
		return NewRedisEngine(deployment), nil
	case "elastic_search":
		return NewElasticSearchEngine(deployment), nil
	default:
		return nil, fmt.Errorf("DB Engine '%s' not supported", deployment.Type)
	}
}
