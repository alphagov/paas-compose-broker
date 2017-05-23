package catalog

import (
	"encoding/json"
	"io"

	"github.com/pivotal-cf/brokerapi"
)

type Catalog struct {
	Services []brokerapi.Service `json:"services"`
}

func New(catalog io.Reader) (*Catalog, error) {
	cp := &Catalog{}
	err := json.NewDecoder(catalog).Decode(cp)
	if err != nil {
		return nil, err
	}
	return cp, nil
}
