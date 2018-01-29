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
			Expect(creds.Hosts).To(Equal([]string{"singlehost.com:10765"}))
			Expect(creds.URI).To(Equal("http://user:password@singlehost.com:10765/"))
			Expect(creds.URIs).To(Equal([]string{"http://user:password@singlehost.com:10765/"}))
		})

		It("should parse multiple connection strings", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{
						"http://user:password@host.com:10764/?ssl=true",
						"http://user:password@nothost.com:10361/?ssl=true",
					},
				},
			})
			icreds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).ToNot(HaveOccurred())
			creds := icreds.(*ElasticSearchCredentials)
			Expect(creds.Username).To(Equal("user"))
			Expect(creds.Password).To(Equal("password"))
			Expect(creds.Hosts).To(Equal([]string{
				"host.com:10764",
				"nothost.com:10361",
			}))
			Expect(creds.URI).To(Equal("http://user:password@host.com:10764/"))
			Expect(creds.URIs).To(Equal([]string{
				"http://user:password@host.com:10764/",
				"http://user:password@nothost.com:10361/",
			}))
		})

		It("should parse any protocol schema and pass it to the URI", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"potato://user:password@singlehost.com:10765/?ssl=true"},
				},
			})
			icreds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).ToNot(HaveOccurred())
			creds := icreds.(*ElasticSearchCredentials)
			Expect(creds.URI).To(Equal("potato://user:password@singlehost.com:10765/"))
			Expect(creds.URIs).To(Equal([]string{"potato://user:password@singlehost.com:10765/"}))
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
			Expect(creds.Hosts).To(Equal([]string{"cluster-name-c00.2.compose.direct:10764"}))
			Expect(creds.URI).To(Equal("http://user:password@cluster-name-c00.2.compose.direct:10764/"))
			Expect(creds.URIs).To(Equal([]string{"http://user:password@cluster-name-c00.2.compose.direct:10764/"}))
		})

		It("should fail to parse empty connection string", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{""},
				},
			})
			creds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).To(HaveOccurred())
			Expect(creds).To(BeNil())
		})

		It("fails to parse a username with invalid encoding", func() {
			engine = NewElasticSearchEngine(&composeapi.Deployment{
				Connection: composeapi.ConnectionStrings{
					Direct: []string{"http://us%er:password@cluster-name-c002.compose.direct:10764/?ssl=true"},
				},
			})
			creds, err := engine.GenerateCredentials("inst1", "bind1")
			Expect(err).To(HaveOccurred())
			Expect(creds).To(BeNil())
		})
	})
})
