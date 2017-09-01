package dbengine

import (
	"os"

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
		dbEngine *MongoEngine
	)

	BeforeEach(func() {
		dbEngine = &MongoEngine{}
	})

	Context("ParseConnectionString()", func() {
		It("should successfully ParseConnectionString() with more than one host", func() {
			deployment := composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@host.com:10764,nothost.com:10361/examples?ssl=true"},
				},
			}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Name).To(Equal("examples"))
			Expect(creds.Username).To(Equal("user"))
		})

		It("should successfully ParseConnectionString() with a single host", func() {
			deployment := composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@host.com:10764/examples?ssl=true"},
				},
			}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Name).To(Equal("examples"))
			Expect(creds.Username).To(Equal("user"))
		})

		It("should fail to ParseConnectionString() due to an invalid url format", func() {
			deployment := composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"%gh&%ij"},
				},
			}

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
		It("should fail to call for CreateUser() due to non-existing session", func() {
			deployment := composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{},
				},
			}
			creds, err := dbEngine.CreateUser("test-case-instance-id", "test-case-binding-id", &deployment)

			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})
	})
})
