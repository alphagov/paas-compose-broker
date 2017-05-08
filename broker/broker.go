package broker

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/config"
	"github.com/pivotal-cf/brokerapi"
)

type Broker struct {
	Config  *config.Config
	Catalog *catalog.Catalog
	Logger  *lager.Logger
}

func New(config *config.Config, catalog *catalog.Catalog, logger *lager.Logger) *Broker {
	return &Broker{
		Config:  config,
		Catalog: catalog,
		Logger:  logger,
	}
}

func (b *Broker) Services(context context.Context) []brokerapi.Service {
	return b.Catalog.Services
}

func (b *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("%s", "Can't provision an instance")
}

func (b *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("%s", "Can't deprovision an instance")
}

func (b *Broker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	return brokerapi.Binding{}, fmt.Errorf("%s", "Can't bind an instance")
}

func (b *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	return fmt.Errorf("%s", "Can't unbind an instance")
}

func (b *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, fmt.Errorf("%s", "Can't update an instance")
}

func (b *Broker) LastOperation(context context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, fmt.Errorf("%s", "Can't check last operation")
}
