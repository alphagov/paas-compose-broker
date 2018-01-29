package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	mgo "gopkg.in/mgo.v2"
)

var _ = Describe("MongoDB", func() {

	Describe("converting a dialInfo to Credentials", func() {
		It("converts a single-host dialInfo string", func() {
			dialInfo, err := mgo.ParseURL("mongodb://user:password@test-c00.2.compose.direct:17445/compose?authSource=admin")
			Expect(err).NotTo(HaveOccurred())

			creds := dialInfoToCredentials(dialInfo, "mylovelycertificate")
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Hosts).To(Equal([]string{"test-c00.2.compose.direct:17445"}))
			Expect(creds.Name).To(Equal("compose"))
			Expect(creds.URI).To(Equal("mongodb://user:password@test-c00.2.compose.direct:17445/compose"))
			Expect(creds.CACertificateBase64).To(Equal("mylovelycertificate"))
		})

		It("converts multi-host dialInfo to Credentials", func() {
			dialInfo, err := mgo.ParseURL("mongodb://user:password@test-c00.2.compose.direct:17445,test-c00.0.compose.direct:17446/compose?authSource=admin")
			Expect(err).NotTo(HaveOccurred())

			creds := dialInfoToCredentials(dialInfo, "mylovelycertificate")
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Hosts).To(Equal([]string{
				"test-c00.2.compose.direct:17445",
				"test-c00.0.compose.direct:17446",
			}))
			Expect(creds.Name).To(Equal("compose"))
			Expect(creds.URI).To(Equal("mongodb://user:password@test-c00.2.compose.direct:17445,test-c00.0.compose.direct:17446/compose"))
			Expect(creds.CACertificateBase64).To(Equal("mylovelycertificate"))
		})
	})

	Context("getMasterDialInfo", func() {
		It("sets a timeout", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445/compose?authSource=admin&ssl=true"},
				},
			})
			dialInfo, err := engine.getMasterDialInfo()
			Expect(err).NotTo(HaveOccurred())
			Expect(dialInfo.Timeout).To(BeNumerically(">", 0))
		})

		It("sets a custom DialServer function, to enable TLS", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445/compose?authSource=admin&ssl=true"},
				},
			})
			dialInfo, err := engine.getMasterDialInfo()
			Expect(err).NotTo(HaveOccurred())
			Expect(dialInfo.DialServer).NotTo(BeNil())
		})

		It("should parse single-host master connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445/compose?authSource=admin&ssl=true"},
				},
			})
			dialInfo, err := engine.getMasterDialInfo()
			Expect(err).ToNot(HaveOccurred())
			Expect(dialInfo.Username).To(Equal("user"))
			Expect(dialInfo.Password).To(Equal("password"))
			Expect(dialInfo.Addrs).To(Equal([]string{"test-c00.2.compose.direct:17445"}))
			Expect(dialInfo.Database).To(Equal("compose"))
			Expect(dialInfo.Source).To(Equal("admin"))
		})

		It("should parse multi-host master connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445,test-c00.0.compose.direct:17446/compose?authSource=admin&ssl=true"},
				},
			})
			dialInfo, err := engine.getMasterDialInfo()
			Expect(err).ToNot(HaveOccurred())
			Expect(dialInfo.Username).To(Equal("user"))
			Expect(dialInfo.Password).To(Equal("password"))
			Expect(dialInfo.Addrs).To(Equal([]string{
				"test-c00.2.compose.direct:17445",
				"test-c00.0.compose.direct:17446",
			}))
			Expect(dialInfo.Database).To(Equal("compose"))
			Expect(dialInfo.Source).To(Equal("admin"))
		})

		It("should parse if there is no auth source", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"mongodb://user:password@test-c00.2.compose.direct:17445,test-c00.0.compose.direct:17445/compose?ssl=true"},
				},
			})
			dialInfo, err := engine.getMasterDialInfo()
			Expect(err).ToNot(HaveOccurred())
			Expect(dialInfo.Username).To(Equal("user"))
			Expect(dialInfo.Password).To(Equal("password"))
			Expect(dialInfo.Addrs).To(Equal([]string{
				"test-c00.2.compose.direct:17445",
				"test-c00.0.compose.direct:17445",
			}))
			Expect(dialInfo.Database).To(Equal("compose"))
			Expect(dialInfo.Source).To(Equal(""))
		})

		It("should fail to parse invalid master connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"%gh&%ij"},
				},
			})
			_, err := engine.getMasterDialInfo()
			Expect(err).To(HaveOccurred())
		})

		It("should fail to parse empty connection string", func() {
			engine := NewMongoEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{""},
				},
			})
			dialInfo, err := engine.getMasterDialInfo()
			Expect(err).Should(HaveOccurred())
			Expect(dialInfo).To(BeNil())
		})

	})

})
