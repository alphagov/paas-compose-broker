package broker

import (
	"errors"

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

		It("returns the deployment if present", func() {
			d, err := findDeployment(fakeComposeClient, "two")
			Expect(err).NotTo(HaveOccurred())
			Expect(d.ID).To(Equal("2345"))
			Expect(d.Name).To(Equal("two"))
		})

		It("returns a errDeploymentNotFound if the deployment doesn't exist", func() {
			_, err := findDeployment(fakeComposeClient, "non-existent")
			Expect(err).To(Equal(errDeploymentNotFound))
		})

		It("returns all other errors", func() {
			fakeComposeClient.GlobalError = errors.New("computer says no")
			_, err := findDeployment(fakeComposeClient, "one")
			Expect(err).To(Equal(fakeComposeClient.GlobalError))
		})
	})

	Describe("makeOperationData", func() {
		It("can make operation data JSON", func() {
			operationData, err := makeOperationData("expected_type", "123")
			Expect(err).ToNot(HaveOccurred())
			Expect(operationData).To(Equal(`{"recipe_id":"123","type":"expected_type"}`))
		})
	})

	Describe("makeInstanceName", func() {
		It("can make an instance name", func() {
			instanceName, err := makeInstanceName("test", "15e332e8-4afa-4c41-82a3-f44b18eba448")
			Expect(err).ToNot(HaveOccurred())
			Expect(instanceName).To(Equal("test-15e332e8-4afa-4c41-82a3-f44b18eba448"))
		})

		It("can trim spaces from dbprefix", func() {
			instanceName, err := makeInstanceName(" trim-spaces ", "0f38f9c2-085c-41ec-87bf-e38b72f7fdaa")
			Expect(err).ToNot(HaveOccurred())
			Expect(instanceName).To(Equal("trim-spaces-0f38f9c2-085c-41ec-87bf-e38b72f7fdaa"))
		})
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
