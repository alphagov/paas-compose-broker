package integration_test

import (
	"github.com/alphagov/paas-compose-broker/broker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("Broker Compose Integration", func() {

	Context("MongoDB", func() {

		BeforeEach(func() {
			if skipIntegrationTests {
				Skip("SKIP_COMPOSE_API_TESTS is set, skipping tests against real Compose API")
			}
		})

		It("should support the full instance lifecycle", func() {

			const (
				mongoServiceID = "36f8bf47-c9e7-46d9-880f-5dfc838d05cb"
				mongoPlanID    = "fdfd4fc1-ce69-451c-a436-c2e2795b9abe"
			)

			var (
				service *helper.ServiceHelper
				binding *helper.BindingData
				conn    *mgo.Session
			)

			By("initializing service from catalog", func() {
				service = helper.NewService(mongoServiceID, mongoPlanID)
			})

			By("provisioning a service", func() {
				service.Provision()
			})

			defer By("deprovisioning the service", func() {
				service.Deprovision()
			})

			By("binding a resource to the service", func() {
				binding = service.Bind()
			})

			defer By("unbinding the service", func() {
				service.Unbind(binding.ID)
			})

			By("connecting to the service", func() {
				var err error
				conn, err = broker.MongoConnection(binding.Credentials.URI, binding.Credentials.CACertificateBase64)
				Expect(err).ToNot(HaveOccurred())
				err = conn.Ping()
				Expect(err).ToNot(HaveOccurred())
			})

			defer By("disconnecting from the service", func() {
				conn.Close()
			})

			By("ensuring binding credentials allow writing data", func() {
				db := conn.DB(binding.Credentials.Name)
				err := db.C("people").Insert(struct {
					Name  string
					Phone string
				}{
					Name:  "John Jones",
					Phone: "+447777777777",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring binding credentials allow reading data", func() {
				db := conn.DB(binding.Credentials.Name)
				result := struct {
					Name  string
					Phone string
				}{}
				err := db.C("people").Find(bson.M{"name": "John Jones"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Name).To(Equal("John Jones"))
				Expect(result.Phone).To(Equal("+447777777777"))
			})

			By("ensuring binding credentials are not for 'admin' user", func() {
				Expect(binding.Credentials.Username).ToNot(Equal("admin"))
			})

			By("ensuring binding credentials are not for 'admin' database", func() {
				Expect(binding.Credentials.Name).ToNot(Equal("admin"))
			})

			By("ensuring binding credentials disallow writing to 'admin' db", func() {
				db := conn.DB("admin")
				err := db.C("bad_people").Insert(struct {
					Name string
				}{
					Name: "Major Kong",
				})
				Expect(err).To(HaveOccurred())
			})

			By("ensuring binding credentials disallow listing 'admin' collections", func() {
				db := conn.DB("admin")
				_, err := db.CollectionNames()
				Expect(err).To(HaveOccurred())
			})

			By("ensuring binding credentials disallow modifying user permissions", func() {
				db := conn.DB(binding.Credentials.Name)
				err := db.UpsertUser(&mgo.User{
					Username: binding.Credentials.Username,
					Password: binding.Credentials.Password,
					Roles:    []mgo.Role{mgo.RoleDBAdminAny},
				})
				Expect(err).To(HaveOccurred())
			})

			By("ensuring binding credentials disallow creating users", func() {
				db := conn.DB(binding.Credentials.Name)
				roles_to_try := []mgo.Role{mgo.RoleReadWrite, mgo.RoleDBAdmin, mgo.RoleReadWriteAny, mgo.RoleDBAdminAny}
				for _, role_to_try := range roles_to_try {
					err := db.UpsertUser(&mgo.User{
						Username: "new_user_they_should_not_be_able_to_create_" + string(role_to_try),
						Password: "zoomzoom" + string(role_to_try),
						Roles:    []mgo.Role{role_to_try},
					})
					Expect(err).To(HaveOccurred())
				}
			})

			By("ensuring binding credentials disallow creating databases", func() {
				db := conn.DB("a_new_db_they_should_not_be_able_to_use")
				err := db.C("bad_people").Insert(struct {
					Name  string
					Phone string
				}{
					Name:  "Major Kong",
					Phone: "+1166666666666",
				})
				Expect(err).To(HaveOccurred())
			})

			By("ensuring each binding returns unique credentials", func() {
				binding2 := service.Bind()
				defer service.Unbind(binding2.ID)
				Expect(binding2.Credentials.Username).ToNot(Equal(binding.Credentials.Username))
				Expect(binding2.Credentials.Password).ToNot(Equal(binding.Credentials.Password))
			})

			By("ensuring each binding returns same database name", func() {
				binding2 := service.Bind()
				defer service.Unbind(binding2.ID)
				Expect(binding2.Credentials.Name).To(Equal(binding.Credentials.Name))
			})

			By("ensuring unbind then rebind to same appID rotates credentials", func() {
				binding := service.Bind() // FIXME/chrisfarms: pass appID to ensure Bind is not basing credentials on appID?
				username := binding.Credentials.Username
				password := binding.Credentials.Password
				name := binding.Credentials.Name
				service.Unbind(binding.ID)
				rebinding := service.Bind()
				defer service.Unbind(rebinding.ID)
				Expect(rebinding.Credentials.Username).ToNot(Equal(username))
				Expect(rebinding.Credentials.Password).ToNot(Equal(password))
				Expect(rebinding.Credentials.Name).To(Equal(name))
			})

			By("ensuring second binding credentials can read previously inserted data", func() {
				binding2 := service.Bind()
				defer service.Unbind(binding2.ID)
				conn2, err := broker.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
				Expect(err).ToNot(HaveOccurred())
				defer conn2.Close()
				people := conn2.DB(binding.Credentials.Name).C("people")
				result := struct {
					Name  string
					Phone string
				}{}
				err = people.Find(bson.M{"name": "John Jones"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Phone).To(Equal("+447777777777"))
			})

			By("ensuring second binding credentials can update previously inserted data", func() {
				binding2 := service.Bind()
				defer service.Unbind(binding2.ID)
				conn2, err := broker.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
				Expect(err).ToNot(HaveOccurred())
				defer conn2.Close()
				people := conn2.DB(binding.Credentials.Name).C("people")
				err = people.Update(bson.M{"name": "John Jones"}, bson.M{"$set": bson.M{"name": "Jane Jones"}})
				Expect(err).ToNot(HaveOccurred())
				result := struct {
					Name  string
					Phone string
				}{}
				err = people.Find(bson.M{"name": "Jane Jones"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Phone).To(Equal("+447777777777"))
			})

			By("ensuring second binding credentials can insert data", func() {
				binding2 := service.Bind()
				defer service.Unbind(binding2.ID)
				conn2, err := broker.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
				Expect(err).ToNot(HaveOccurred())
				defer conn2.Close()
				db := conn2.DB(binding.Credentials.Name)
				err = db.C("people").Insert(struct {
					Name  string
					Phone string
				}{
					Name:  "Tim Timmis",
					Phone: "+17734573777",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring revoked bindings disallow connecting", func() {
				revokedBinding := service.Bind()
				service.Unbind(revokedBinding.ID)
				_, err := broker.MongoConnection(revokedBinding.Credentials.URI, revokedBinding.Credentials.CACertificateBase64)
				Expect(err).To(HaveOccurred())
			})

			By("ensuring binding credentials allow deleting data", func() {
				db := conn.DB(binding.Credentials.Name)
				err := db.C("people").Remove(bson.M{"name": "Tim Timmis"})
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring binding credentials allow deleting collections", func() {
				db := conn.DB(binding.Credentials.Name)
				err := db.C("people").DropCollection()
				Expect(err).ToNot(HaveOccurred())
			})

		})

	})

})
