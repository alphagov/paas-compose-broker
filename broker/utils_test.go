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
		It("returns the deployment if present", func() {
			fakeComposeClient := &fakes.FakeClient{}
			deployment := &composeapi.Deployment{ID: "2345", Name: "two"}
			fakeComposeClient.GetDeploymentByNameReturns(deployment, []error{})

			Expect(findDeployment(fakeComposeClient, "two")).To(Equal(deployment))
		})

		It("returns a errDeploymentNotFound if the deployment doesn't exist", func() {
			fakeComposeClient := &fakes.FakeClient{}
			fakeComposeClient.GetDeploymentByNameReturns(nil, []error{errors.New("deployment not found")})

			_, err := findDeployment(fakeComposeClient, "non-existent")

			Expect(err).To(Equal(errDeploymentNotFound))
		})

		It("returns all other errors", func() {
			fakeComposeClient := &fakes.FakeClient{}
			fakeComposeClient.GetDeploymentByNameReturns(nil, []error{errors.New("computer says no")})

			_, err := findDeployment(fakeComposeClient, "one")

			Expect(err).To(MatchError("computer says no"))
		})
	})

	Describe("makeOperationData", func() {
		It("can make operation data JSON", func() {
			operationData, err := makeOperationData("expected_type", "123", []string{"1.1.1.1"})
			Expect(err).ToNot(HaveOccurred())
			Expect(operationData).To(Equal(`{"type":"expected_type","recipe_id":"123","whitelist_recipe_ids":["1.1.1.1"]}`))
		})
	})

	Describe("makeInstanceName", func() {
		It("can make an instance name", func() {
			instanceName, err := MakeInstanceName("test", "15e332e8-4afa-4c41-82a3-f44b18eba448")
			Expect(err).ToNot(HaveOccurred())
			Expect(instanceName).To(Equal("test-15e332e8-4afa-4c41-82a3-f44b18eba448"))
		})

		It("can trim spaces from dbprefix", func() {
			instanceName, err := MakeInstanceName(" trim-spaces ", "0f38f9c2-085c-41ec-87bf-e38b72f7fdaa")
			Expect(err).ToNot(HaveOccurred())
			Expect(instanceName).To(Equal("trim-spaces-0f38f9c2-085c-41ec-87bf-e38b72f7fdaa"))
		})
	})
})
