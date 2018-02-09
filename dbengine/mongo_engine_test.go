package dbengine_test

import (
	"errors"
	"reflect"

	"github.com/alphagov/paas-compose-broker/dbengine"
	"github.com/alphagov/paas-compose-broker/dbengine/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MongoDB Engine", func() {

	Describe("GetDatabaseName()", func() {
		Context("if the default database already exists", func() {
			It("should return the default database name", func() {
				mongoEngine := dbengine.MongoEngine{}
				session := &fakes.FakeMongoSession{}
				session.RunStub = createMongoSessionRunStub("admin", "default")
				Expect(mongoEngine.GetDatabaseName(session, "default")).To(Equal("default"))
			})
		})

		Context("if a database was already created with the old name (db_*)", func() {
			It("should return the existing database name", func() {
				mongoEngine := dbengine.MongoEngine{}
				session := &fakes.FakeMongoSession{}
				session.RunStub = createMongoSessionRunStub("admin", "db_123")
				Expect(mongoEngine.GetDatabaseName(session, "default")).To(Equal("db_123"))
			})
		})

		Context("if there is no database created for the user yet", func() {
			It("should return the default database name", func() {
				mongoEngine := dbengine.MongoEngine{}
				session := &fakes.FakeMongoSession{}
				session.RunStub = createMongoSessionRunStub("admin")
				Expect(mongoEngine.GetDatabaseName(session, "default")).To(Equal("default"))
			})
		})

		Context("if MongoDB returns with an error", func() {
			It("should return with an error", func() {
				mongoEngine := dbengine.MongoEngine{}
				session := &fakes.FakeMongoSession{}
				session.RunStub = func(cmd interface{}, result interface{}) error {
					return errors.New("some error")
				}
				_, err := mongoEngine.GetDatabaseName(session, "default")
				Expect(err).To(HaveOccurred())
			})
		})

	})

})

func createMongoSessionRunStub(databaseNames ...string) func(cmd interface{}, result interface{}) error {
	res := &dbengine.DatabaseNames{
		Databases: []struct {
			Name  string
			Empty bool
		}{},
	}
	for _, dbName := range databaseNames {
		res.Databases = append(res.Databases, struct {
			Name  string
			Empty bool
		}{
			Name: dbName,
		})
	}
	return func(cmd interface{}, result interface{}) error {
		reflect.ValueOf(result).Elem().Set(reflect.ValueOf(*res))
		return nil
	}
}
