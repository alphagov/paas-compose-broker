package fakes

import "github.com/alphagov/paas-compose-broker/dbengine"

type FakeProvider struct {
	DBEngine      dbengine.DBEngine
	DBEngineError error
}

func (f FakeProvider) GetDBEngine(engine string) (dbengine.DBEngine, error) {
	return &FakeDBEngine{}, nil
}
