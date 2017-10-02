package catalog

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pivotal-cf/brokerapi"
)

type Service struct {
	Plans []*Plan `json:"plans"`
	brokerapi.Service
}

type Plan struct {
	Compose ComposeConfig `json:"compose"`
	brokerapi.ServicePlan
}

type ComposeConfig struct {
	Units        int    `json:"units"`
	DatabaseType string `json:"databaseType"`
	CacheMode    bool   `json:"cacheMode"`
}

type Catalog struct {
	Services []*Service `json:"services"`
}

func (c *Catalog) GetService(id string) (*Service, error) {
	for _, service := range c.Services {
		if service.ID == id {
			return service, nil
		}
	}

	return nil, fmt.Errorf("service %v: not found", id)
}

func (s *Service) GetPlan(id string) (*Plan, error) {
	for _, plan := range s.Plans {
		if plan.ID == id {
			return plan, nil
		}
	}

	return nil, fmt.Errorf("plan %v: not found", id)
}

func Load(input io.Reader) (*Catalog, error) {
	var c Catalog
	if err := json.NewDecoder(input).Decode(&c); err != nil {
		return nil, err
	}
	for _, s := range c.Services {
		for _, p := range s.Plans {
			s.Service.Plans = append(s.Service.Plans, p.ServicePlan)
		}
	}
	return &c, nil
}
