package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pivotal-cf/brokerapi"
)

type ComposeCatalog struct {
	Catalog      Catalog
	ComposeUnits ComposeUnits
}

// Catalog is an upstream Catalog struct
type Catalog struct {
	Services []brokerapi.Service `json:"services"`
}

type ComposeUnits struct {
	Services []Service `json:"services"`
}

type Service struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Plans []ServicePlan `json:"plans"`
}

type ServicePlan struct {
	ID       string              `json:"id"`
	Metadata ServicePlanMetadata `json:"metadata,omitempty"`
}

type ServicePlanMetadata struct {
	Units int `json:"units,omitempty"`
}

func (c *ComposeCatalog) GetService(id string) (*Service, error) {
	for _, service := range c.ComposeUnits.Services {
		if service.ID == id {
			return &service, nil
		}
	}

	return nil, fmt.Errorf("service: not found")
}

func (s *Service) GetPlan(id string) (*ServicePlan, error) {
	for _, plan := range s.Plans {
		if plan.ID == id {
			return &plan, nil
		}
	}

	return nil, fmt.Errorf("plan: not found")
}

func Load(input io.Reader) (*ComposeCatalog, error) {
	buf, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	c := &ComposeCatalog{}

	err = json.Unmarshal(buf, &c.Catalog)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(buf, &c.ComposeUnits)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *ComposeCatalog) Validate() error {
	return nil
}
