package integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/alphagov/paas-compose-broker/integration_tests/helper"
)

var _ = Describe("Broker Compose Integration", func() {

	Context("MongoDB", func() {

		const (
			mongoServiceID = "36f8bf47-c9e7-46d9-880f-5dfc838d05cb"
			mongoPlanID    = "fdfd4fc1-ce69-451c-a436-c2e2795b9abe"
		)

		var (
			service                 *helper.ServiceHelper
			instanceID, instanceID2 string
			binding, binding2       *helper.BindingData
			conn, conn2             *mgo.Session
			appID                   string
		)

		BeforeEach(func() {
			if skipIntegrationTests {
				Skip("SKIP_COMPOSE_API_TESTS is set, skipping tests against real Compose API")
			}

			appID = helper.NewUUID()
		})

		It("should support the full instance lifecycle", func() {

			By("initializing service from catalog", func() {
				service = helper.NewService(mongoServiceID, mongoPlanID, []string{})
			})

			By("provisioning a service", func() {
				instanceID = service.Provision(map[string]interface{}{})
			})

			defer By("deprovisioning the service", func() {
				service.Deprovision(instanceID)
			})

			By("binding a resource to the service", func() {
				binding = service.Bind(instanceID, appID)
			})

			defer By("unbinding the service", func() {
				service.Unbind(instanceID, binding.ID)
			})

			By("connecting to the service", func() {
				var err error
				conn, err = helper.MongoConnection(binding.Credentials.URI, binding.Credentials.CACertificateBase64)
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
				binding2 := service.Bind(instanceID, appID)
				defer service.Unbind(instanceID, binding2.ID)
				Expect(binding2.Credentials.Username).ToNot(Equal(binding.Credentials.Username))
				Expect(binding2.Credentials.Password).ToNot(Equal(binding.Credentials.Password))
			})

			By("ensuring each binding returns same database name", func() {
				binding2 := service.Bind(instanceID, appID)
				defer service.Unbind(instanceID, binding2.ID)
				Expect(binding2.Credentials.Name).To(Equal(binding.Credentials.Name))
			})

			By("ensuring unbind then rebind to same appID rotates credentials", func() {
				binding := service.Bind(instanceID, appID)
				username := binding.Credentials.Username
				password := binding.Credentials.Password
				name := binding.Credentials.Name
				service.Unbind(instanceID, binding.ID)
				rebinding := service.Bind(instanceID, appID)
				defer service.Unbind(instanceID, rebinding.ID)
				Expect(rebinding.Credentials.Username).ToNot(Equal(username))
				Expect(rebinding.Credentials.Password).ToNot(Equal(password))
				Expect(rebinding.Credentials.Name).To(Equal(name))
			})

			By("ensuring second binding credentials can read previously inserted data", func() {
				binding2 := service.Bind(instanceID, appID)
				defer service.Unbind(instanceID, binding2.ID)
				conn2, err := helper.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
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
				binding2 := service.Bind(instanceID, appID)
				defer service.Unbind(instanceID, binding2.ID)
				conn2, err := helper.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
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
				binding2 := service.Bind(instanceID, appID)
				defer service.Unbind(instanceID, binding2.ID)
				conn2, err := helper.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
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
				revokedBinding := service.Bind(instanceID, appID)
				service.Unbind(instanceID, revokedBinding.ID)
				_, err := helper.MongoConnection(revokedBinding.Credentials.URI, revokedBinding.Credentials.CACertificateBase64)
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

			By("inserting data for backup test", func() {
				conn.SetSafe(&mgo.Safe{
					W:        3,
					FSync:    true,
					WTimeout: 30 * 1000,
				})
				defer conn.SetSafe(&mgo.Safe{})

				db := conn.DB(binding.Credentials.Name)
				err := db.C("backup").Insert(struct {
					Name  string
					Phone string
				}{
					Name:  "Backup Bob",
					Phone: "+44123123123",
				})
				Expect(err).ToNot(HaveOccurred())
			})

			By("ensuring we have a backup", func() {
				time.Sleep(1 * time.Minute)

				deploymentName := fmt.Sprintf("%s-%s", service.Cfg.DBPrefix, instanceID)
				helper.CreateBackup(service.ComposeClient, deploymentName)
			})

			By("creating a new service instance from backup", func() {
				instanceID2 = service.Provision(map[string]interface{}{"restore_from_latest_snapshot_of": instanceID})
			})

			defer By("deprovisioning the service created from backup", func() {
				service.Deprovision(instanceID2)
			})

			By("binding the app to the service created from backup", func() {
				binding2 = service.Bind(instanceID2, appID)
			})

			defer By("unbinding the service created from backup", func() {
				service.Unbind(instanceID2, binding2.ID)
			})

			By("connecting to the service created from backup", func() {
				var err error
				conn2, err = helper.MongoConnection(binding2.Credentials.URI, binding2.Credentials.CACertificateBase64)
				Expect(err).ToNot(HaveOccurred())
				err = conn2.Ping()
				Expect(err).ToNot(HaveOccurred())
			})

			defer By("disconnecting from the service created from backup", func() {
				conn2.Close()
			})

			By("ensuring the service created from backup contains the data from the other instance", func() {
				db := conn2.DB(binding2.Credentials.Name)
				result := struct {
					Name  string
					Phone string
				}{}
				err := db.C("backup").Find(bson.M{"name": "Backup Bob"}).One(&result)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Name).To(Equal("Backup Bob"))
				Expect(result.Phone).To(Equal("+44123123123"))
			})

		})

	})

})
