package dbengine

import (
	"fmt"
	"os"

	mgo "gopkg.in/mgo.v2"

	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func createDB(credentials Credentials, dbname string) error {
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

var _ = Describe("Mongo DBEngine", func() {
	var (
		dbEngine        *MongoEngine
		deployment      composeapi.Deployment
		rootCredentials Credentials
	)

	BeforeEach(func() {
		rootCredentials = Credentials{
			Host:     getEnvOrDefault("MONGO_INITDB_HOSTNAME", "localhost"),
			Name:     getEnvOrDefault("MONGO_INITDB_DATABASE", "mydb_test"),
			Password: getEnvOrDefault("MONGO_INITDB_ROOT_PASSWORD", "must_not_be_empty"),
			Port:     getEnvOrDefault("MONGO_INITDB_PORT", "27017"),
			Username: getEnvOrDefault("MONGO_INITDB_ROOT_USERNAME", "travis"),
		}
		rootCredentials.URI = fmt.Sprintf(
			"mongodb://%s:%s@%s:%s/%s",
			rootCredentials.Username,
			rootCredentials.Password,
			rootCredentials.Host,
			rootCredentials.Port,
			rootCredentials.Name,
		)

		session, err := mgo.Dial(fmt.Sprintf("%s:%s", rootCredentials.Host, rootCredentials.Port))
		Expect(err).ShouldNot(HaveOccurred())

		dbEngine = &MongoEngine{session}
		deployment = composeapi.Deployment{
			Connection: composeapi.ConnectionStrings{
				Direct: []string{rootCredentials.URI},
			},
		}
	})

	Context("ParseConnectionString()", func() {
		It("should successfully ParseConnectionString() with more than one host", func() {
			deployment.Connection.Direct = []string{"mongodb://user:password@host.com:10764,nothost.com:10361/examples?ssl=true"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Name).To(Equal("examples"))
			Expect(creds.Username).To(Equal("user"))
		})

		It("should successfully ParseConnectionString() with a single host", func() {
			deployment.Connection.Direct = []string{"mongodb://user:password@host.com:10764/examples?ssl=true"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Name).To(Equal("examples"))
			Expect(creds.Username).To(Equal("user"))
		})

		It("should fail to ParseConnectionString() due to an invalid url format", func() {
			deployment.Connection.Direct = []string{"%gh&%ij"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})

		It("should fail to ParseConnectionString() due to lack of deployment", func() {
			creds, err := dbEngine.ParseConnectionString(nil)
			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})
	})

	Describe("requiring the database connection", func() {
		It("should call for CreateUser() successfully", func() {
			creds, err := dbEngine.CreateUser("test-case-instance-id", "test-case-binding-id", &deployment)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal(rootCredentials.Host))
			Expect(creds.Username).NotTo(Equal(rootCredentials.Username))
			Expect(creds.Password).NotTo(Equal(rootCredentials.Password))
		})

		It("should fail to call for CreateUser() due to non-existing session", func() {
			dbEngine.session = nil
			creds, err := dbEngine.CreateUser("test-case-instance-id", "test-case-binding-id", &deployment)

			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})
	})
})
