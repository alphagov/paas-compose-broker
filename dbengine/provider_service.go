package dbengine

import (
	"fmt"
	"strings"
)

type ProviderService struct{}

func NewProviderService() *ProviderService {
	return &ProviderService{}
}

func (p *ProviderService) GetDBEngine(engine string) (DBEngine, error) {
	switch strings.ToLower(engine) {
	case "mongodb":
		return NewMongoEngine(), nil
	case "redis":
		return NewRedisEngine(), nil
	case "elastic_search":
		return NewElasticSearchEngine(), nil
	}

	return nil, fmt.Errorf("DB Engine '%s' not supported", engine)
}
