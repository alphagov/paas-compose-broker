package catalog

import (
	"encoding/json"

	"github.com/pivotal-cf/brokerapi"
)

type Catalog struct {
	Services []brokerapi.Service `json:"services"`
}

func New() *Catalog {
	return &Catalog{}
}

func (c *Catalog) Load(data []byte) error {
	err := json.Unmarshal(data, c)
	if err != nil {
		return err
	}
	return nil
}
