package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ElasticSearch", func() {
	var (
		engine DBEngine
	)

	Context("GenerateCredentials", func() {

		It("should parse single-host master connection string", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"http://user:password@singlehost.com:10765/?ssl=true"},
				},
			})
			icreds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).ToNot(HaveOccurred())
			creds := icreds.(*ElasticSearchCredentials)
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Host).To(Equal("singlehost.com"))
			Expect(creds.Port).To(Equal("10765"))
		})

		It("should parse multi-host master connection string", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"http://user:password@host.com:10764,nothost.com:10361/?ssl=true"},
				},
			})
			icreds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).ToNot(HaveOccurred())
			creds := icreds.(*ElasticSearchCredentials)
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Host).To(Equal("host.com"))
			Expect(creds.Port).To(Equal("10764"))
		})

		It("should fail to parse invalid master connection string", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"%gh&%ij"},
				},
			})
			_, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).To(HaveOccurred())
		})

		It("should workaround incorrect compose hostname", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"http://user:password@cluster-name-c002.compose.direct:10764/?ssl=true"},
				},
			})
			icreds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).ToNot(HaveOccurred())
			creds := icreds.(*ElasticSearchCredentials)
			Expect(creds.Host).To(Equal("cluster-name-c00.2.compose.direct"))
		})

		It("should fail to parse empty connection string", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{""},
				},
			})
			creds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).Should(HaveOccurred())
			Expect(creds).To(BeNil())
		})

	})

})
