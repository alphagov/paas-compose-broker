package catalog

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	catalogJson = `
		{
		  "services": [{
		    "id": "XXXX-XXXX-XXXX-XXXX",
		    "name": "SERVICE_NAME",
		    "plans": [{
		      "id": "YYYY-YYYY-YYYY-YYYY",
		      "name": "PLAN_NAME",
		      "description": "DATABASE_DESCRIPTION",
		      "compose": {
		        "units": 1,
		        "databaseType": "DATABASE_TYPE"
		      }
		    }]
		  }]
		}
	`
)

var _ = Describe("catalog", func() {

	var (
		catalog *Catalog
		err     error
	)

	BeforeEach(func() {
		catalog, err = Load(strings.NewReader(catalogJson))
		Expect(err).ToNot(HaveOccurred())
		Expect(catalog.Services).ToNot(BeEmpty(), "example catalog should have at least one service")
		Expect(catalog.Services[0].Plans).ToNot(BeEmpty(), "example catalog should have at least one plan")
	})

	It("should have a service id set", func() {
		Expect(catalog.Services[0].ID).To(Equal("XXXX-XXXX-XXXX-XXXX"))
	})

	It("should have a service name set", func() {
		Expect(catalog.Services[0].Name).To(Equal("SERVICE_NAME"))
	})

	It("should have a plan with an id set", func() {
		Expect(catalog.Services[0].Plans[0].ID).To(Equal("YYYY-YYYY-YYYY-YYYY"))
	})

	It("should have a plan with a name set", func() {
		Expect(catalog.Services[0].Plans[0].Name).To(Equal("PLAN_NAME"))
	})

	It("should have a plan with compose config set", func() {
		Expect(catalog.Services[0].Plans[0].Compose.Units).To(Equal(1), "expected units set")
		Expect(catalog.Services[0].Plans[0].Compose.DatabaseType).To(Equal("DATABASE_TYPE"), "expected a databaseType set")
	})

	It("should expose the embedded brokerapi.Service type", func() {
		service := catalog.Services[0]
		brokerService := service.Service
		Expect(brokerService).ToNot(BeNil())
		Expect(brokerService.ID).To(Equal(service.ID))
		Expect(brokerService.Name).To(Equal(service.Name))
		Expect(brokerService.Plans).ToNot(BeEmpty())
		plan := service.Plans[0]
		brokerPlan := brokerService.Plans[0]
		Expect(brokerPlan.ID).To(Equal(plan.ID))
		Expect(brokerPlan.Name).To(Equal(plan.Name))
	})

})

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog Suite")
}
