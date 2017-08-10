package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

var _ = Describe("ElasticSearch", func() {
	var (
		credentials Credentials
		dbEngine    *ElasticSearchEngine
		deployment  composeapi.Deployment
	)

	BeforeEach(func() {
		credentials = Credentials{
			URI: getEnvOrDefault("ES_URL", "http://localhost:9200"),
		}

		deployment = composeapi.Deployment{
			Connection: composeapi.ConnectionStrings{
				Direct: []string{credentials.URI},
			},
		}
	})

	Context("Connection Testing", func() {
		BeforeEach(func() {
			dbEngine = &ElasticSearchEngine{credentials: &credentials}
		})

		It("should testConnection() successfully", func() {
			err := dbEngine.testConnection()

			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should fail to testConnection() due to faulty url", func() {
			dbEngine.credentials.URI = "http://localhost:8080"
			err := dbEngine.testConnection()

			Expect(err).Should(HaveOccurred())
		})

		It("should fail to testConnection() due to server error", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponder("GET", "http://localhost:9200",
				httpmock.NewStringResponder(500, ``))

			err := dbEngine.testConnection()

			Expect(err).Should(HaveOccurred())
		})

		It("should fail to testConnection() due to lack of version returned", func() {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponder("GET", "http://localhost:9200",
				httpmock.NewStringResponder(200, `{"name":"x","cluster_name":"xXx","cluster_uuid":"xxx-123","version":{"number":"","build_hash":"x","build_date":"x","build_snapshot":false,"lucene_version":""},"tagline":"You Know, for Search"}`))

			err := dbEngine.testConnection()

			Expect(err).Should(HaveOccurred())
		})
	})

	Context("DB Engine", func() {
		dbEngine = &ElasticSearchEngine{}

		It("should Open() the connection successfully", func() {
			err := dbEngine.Open(&credentials)

			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should fail to Open() the connection due to lack of credentials being provided", func() {
			err := dbEngine.Open(nil)

			Expect(err).Should(HaveOccurred())
		})

		It("should fail to Open() the connection due to invalid URL provided", func() {
			creds := Credentials{URI: "http://localhost:8080"}
			err := dbEngine.Open(&creds)

			Expect(err).Should(HaveOccurred())
		})

		It("should successfully ParseConnectionString() with more than one host", func() {
			deployment.Connection.Direct = []string{"http://user:password@host.com:10764,nothost.com:10361/?ssl=true"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Username).To(Equal("user"))
		})

		It("should successfully ParseConnectionString() with a single host", func() {
			deployment.Connection.Direct = []string{"http://user:password@host.com:10764/?ssl=true"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Username).To(Equal("user"))
		})

		It("should fail to ParseConnectionString() due to an invalid url format", func() {
			deployment.Connection.Direct = []string{"%gh&%ij"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})

		It("should mend incorrect hostnames", func() {
			deployment.Connection.Direct = []string{"http://user:password@cluster-name-c002.compose.direct:10764/?ssl=true"}

			creds, err := dbEngine.ParseConnectionString(&deployment)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds.Host).To(Equal("cluster-name-c00.2.compose.direct"))
		})

		It("should fail to ParseConnectionString() due to lack of deployment", func() {
			creds, err := dbEngine.ParseConnectionString(nil)
			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})

		It("does not need to create users", func() {
			creds, err := dbEngine.CreateUser("instanceID", "bindingID", &deployment)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(creds).NotTo(BeNil())
			Expect(creds.Username).To(BeEmpty())
			Expect(creds.Password).To(BeEmpty())
		})
	})
})
