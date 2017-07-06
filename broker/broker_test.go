package broker_test

import (
	"errors"

	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose/fakes"
	"github.com/alphagov/paas-compose-broker/config"
)

var _ = Describe("Broker", func() {

	Describe("constructing a broker", func() {
		var (
			fakeComposeClient *fakes.FakeComposeClient
			cfg               *config.Config
		)

		BeforeEach(func() {
			fakeComposeClient = fakes.New()
			cfg = &config.Config{}
		})

		Describe("looking up the compose Account ID", func() {
			BeforeEach(func() {
				fakeComposeClient.Account = composeapi.Account{ID: "1234"}
			})

			It("looks up the compose account ID", func() {
				b, err := broker.New(fakeComposeClient, cfg, &catalog.ComposeCatalog{}, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(b.AccountID).To(Equal("1234"))
			})

			It("returns an error if looking up the account fails", func() {
				fakeComposeClient.GlobalError = errors.New("something went wrong")
				_, err := broker.New(fakeComposeClient, cfg, &catalog.ComposeCatalog{}, nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("populating the cluster ID", func() {
			BeforeEach(func() {
				fakeComposeClient.Clusters = []composeapi.Cluster{
					{
						ID:   "1",
						Name: "cluster-one",
					},
					{
						ID:   "2",
						Name: "cluster-two",
					},
				}
			})

			It("leaves the cluster ID blank if no cluster name provided", func() {
				b, err := broker.New(fakeComposeClient, cfg, &catalog.ComposeCatalog{}, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(b.ClusterID).To(BeEmpty())
			})

			It("populates the clusterID using the provided name", func() {
				cfg.ClusterName = "cluster-two"

				b, err := broker.New(fakeComposeClient, cfg, &catalog.ComposeCatalog{}, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(b.ClusterID).To(Equal("2"))
			})

			It("returns an error if the cluster ID can't be looked up", func() {
				cfg.ClusterName = "non-existent"

				_, err := broker.New(fakeComposeClient, cfg, &catalog.ComposeCatalog{}, nil)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
