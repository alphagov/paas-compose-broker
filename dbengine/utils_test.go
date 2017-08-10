package dbengine

import (
	composeapi "github.com/compose/gocomposeapi"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-compose-broker/compose/fakes"
)

var _ = Describe("Broker utility functions", func() {

	Describe("findDeployment", func() {
		var (
			fakeComposeClient *fakes.FakeComposeClient
		)

		BeforeEach(func() {
			fakeComposeClient = fakes.New()
			fakeComposeClient.Deployments = []composeapi.Deployment{
				{ID: "1234", Name: "one"},
				{ID: "2345", Name: "two"},
			}
		})

		Describe("makeUserName", func() {
			It("can make a user name", func() {
				userName := makeUserName("62a334e8-5afa-7c41-92a3-a44b18eba448")
				Expect(userName).To(Equal("user_62a334e8-5afa-7c41-92a3-a44b18eba448"))
			})
		})

		Describe("makeDatabaseName", func() {
			It("can make a database name", func() {
				userName := makeDatabaseName("62a334e8-5afa-7c41-92a3-a44b18eba448")
				Expect(userName).To(Equal("db_62a334e8-5afa-7c41-92a3-a44b18eba448"))
			})
		})

		Describe("makeRandomPassword", func() {
			It("can make a random password", func() {
				By("generating strings of at least a given length", func() {
					randomPassword, err := makeRandomPassword(10)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(randomPassword)).To(BeNumerically(">", 10))

					randomPassword, err = makeRandomPassword(30)
					Expect(err).ToNot(HaveOccurred())
					Expect(len(randomPassword)).To(BeNumerically(">", 30))
				})

				By("generating numerous different values", func() {
					observedPasswords := map[string]bool{}
					for i := 0; i < 100; i++ {
						password, err := makeRandomPassword(30)
						Expect(err).ToNot(HaveOccurred())
						Expect(observedPasswords).To(Not(ContainElement(password)))
						observedPasswords[password] = true
					}
				})
			})
		})
	})
})
