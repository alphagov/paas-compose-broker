package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MongoDB", func() {

	Context("getMasterDialInfo", func() {
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
