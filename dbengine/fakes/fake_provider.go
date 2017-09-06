package fakes

import (
	"fmt"

	"github.com/alphagov/paas-compose-broker/dbengine"
	composeapi "github.com/compose/gocomposeapi"
)

type FakeProvider struct {
	DBEngine      dbengine.DBEngine
	DBEngineError error
}

func (f FakeProvider) GetDBEngine(deployment *composeapi.Deployment) (dbengine.DBEngine, error) {
	switch deployment.Type {
	case "fakedb":
		return &FakeDBEngine{deployment}, nil
	default:
		return nil, fmt.Errorf("FakeProvider does not support type: %s", deployment.Type)
	}
}
