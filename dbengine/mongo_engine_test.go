package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MongoDB", func() {

	Context("getMasterCredentials", func() {

		It("should parse single-host master connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445/compose?authSource=admin&ssl=true"},
				},
			})
			creds, err := engine.getMasterCredentials()
			Expect(err).ToNot(HaveOccurred())
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Host).To(Equal("test-c00.2.compose.direct"))
			Expect(creds.Port).To(Equal("17445"))
			Expect(creds.Name).To(Equal("compose"))
			Expect(creds.AuthSource).To(Equal("admin"))
		})

		It("should parse multi-host master connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445,test-c00.0.compose.direct:17445/compose?authSource=admin&ssl=true"},
				},
			})
			creds, err := engine.getMasterCredentials()
			Expect(err).ToNot(HaveOccurred())
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Host).To(Equal("test-c00.2.compose.direct"))
			Expect(creds.Port).To(Equal("17445"))
			Expect(creds.Name).To(Equal("compose"))
			Expect(creds.AuthSource).To(Equal("admin"))
		})

		It("should parse if there is no auth source", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445,test-c00.0.compose.direct:17445/compose?ssl=true"},
				},
			})
			creds, err := engine.getMasterCredentials()
			Expect(err).ToNot(HaveOccurred())
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Host).To(Equal("test-c00.2.compose.direct"))
			Expect(creds.Port).To(Equal("17445"))
			Expect(creds.Name).To(Equal("compose"))
			Expect(creds.AuthSource).To(Equal(""))
		})

		It("should fail to parse invalid master connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"%gh&%ij"},
				},
			})
			_, err := engine.getMasterCredentials()
			Expect(err).To(HaveOccurred())
		})

		It("should fail to parse empty connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{""},
				},
			})
			creds, err := engine.getMasterCredentials()
			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})

	})

})
