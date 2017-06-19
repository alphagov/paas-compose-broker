package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose"
	"github.com/alphagov/paas-compose-broker/config"
	"github.com/compose/gocomposeapi"
	"github.com/pivotal-cf/brokerapi"
)

const (
	ComposeDatacenter   = "aws:eu-west-1"
	instanceIDLogKey    = "instance-id"
	bindingIDLogKey     = "binding-id"
	detailsLogKey       = "details"
	asyncAllowedLogKey  = "acceptsIncomplete"
	operationDataLogKey = "operation-data-recipe-id"
)

type Broker struct {
	Compose        compose.Client
	Config         *config.Config
	ComposeCatalog *catalog.ComposeCatalog
	Logger         lager.Logger
}

type Credentials struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	URI      string `json:"uri"`
	JDBCURI  string `json:"jdbcuri"`
}

type OperationData struct {
	RecipeID string `json:"recipe_id"`
	Type     string `json:"type"`
}

var composeStatus2State = map[string]brokerapi.LastOperationState{
	"complete": brokerapi.Succeeded,
	"running":  brokerapi.InProgress,
	"waiting":  brokerapi.InProgress,
}

func New(compose compose.Client, config *config.Config, catalog *catalog.ComposeCatalog, logger lager.Logger) (*Broker, error) {
	if compose == nil {
		return nil, fmt.Errorf("broker: composer should not be nil")
	}

	broker := Broker{
		Compose:        compose,
		Config:         config,
		ComposeCatalog: catalog,
		Logger:         logger,
	}

	return &broker, nil
}

func (b *Broker) Services(context context.Context) []brokerapi.Service {
	return b.ComposeCatalog.Catalog.Services
}

func (b *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	b.Logger.Debug("provision", lager.Data{
		instanceIDLogKey:   instanceID,
		detailsLogKey:      details,
		asyncAllowedLogKey: asyncAllowed,
	})

	spec := brokerapi.ProvisionedServiceSpec{
		IsAsync: true,
	}

	if !asyncAllowed {
		return spec, brokerapi.ErrAsyncRequired
	}

	service, err := b.ComposeCatalog.GetService(details.ServiceID)
	if err != nil {
		return spec, err
	}

	plan, err := service.GetPlan(details.PlanID)
	if err != nil {
		return spec, err
	}

	instanceName, err := makeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return spec, err
	}

	params := composeapi.DeploymentParams{
		Name:         instanceName,
		AccountID:    b.Config.AccountID,
		Datacenter:   ComposeDatacenter,
		DatabaseType: service.Name,
		Units:        plan.Metadata.Units,
		SSL:          true,
	}

	deployment, errs := b.Compose.CreateDeployment(params)
	if len(errs) > 0 {
		return spec, squashErrors(errs)
	}

	operationData, err := makeOperationData("provision", deployment.ProvisionRecipeID)
	if err != nil {
		return spec, err
	}

	spec.OperationData = operationData

	return spec, nil
}

func (b *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	b.Logger.Debug("deprovision", lager.Data{
		instanceIDLogKey:   instanceID,
		detailsLogKey:      details,
		asyncAllowedLogKey: asyncAllowed,
	})

	spec := brokerapi.DeprovisionServiceSpec{
		IsAsync: true,
	}

	if !asyncAllowed {
		return spec, brokerapi.ErrAsyncRequired
	}

	instanceName, err := makeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return spec, err
	}

	deployment, err := findDeployment(b.Compose, instanceName)
	if err != nil {
		return spec, brokerapi.ErrInstanceDoesNotExist
	}

	recipe, errs := b.Compose.DeprovisionDeployment(deployment.ID)
	if len(errs) > 0 {
		return spec, squashErrors(errs)
	}

	operationData, err := makeOperationData("deprovision", recipe.ID)
	if err != nil {
		return spec, err
	}

	spec.OperationData = operationData

	return spec, nil
}

func (b *Broker) Bind(context context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	b.Logger.Debug("bind", lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
		detailsLogKey:    details,
	})

	binding := brokerapi.Binding{}

	instanceName, err := makeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return binding, err
	}

	deploymentMeta, err := findDeployment(b.Compose, instanceName)
	if err != nil {
		return binding, err
	}

	deployment, errs := b.Compose.GetDeployment(deploymentMeta.ID)
	if len(errs) > 0 {
		return binding, squashErrors(errs)
	}
	if deployment.Connection.Direct == nil || len(deployment.Connection.Direct) < 1 {
		return binding, fmt.Errorf("failed to get connection string")
	}
	bindingURL, err := url.Parse(deployment.Connection.Direct[0])
	if err != nil {
		return binding, err
	}
	password, _ := bindingURL.User.Password()

	// FIXME: Follow up story should fix mongo connection string handling.
	// Right now we are hardcoding first host from the comma delimited list that Compose provides.
	// url.Parse() parses mongo connection string wrongly and doesn't return an error
	// so url.Port() returns port like "18899,aws-eu-west-1-portal.7.dblayer.com:18899"
	port := strings.Split(bindingURL.Port(), ",")

	binding.Credentials = Credentials{
		Host:     bindingURL.Hostname(),
		Port:     port[0],
		Name:     bindingURL.RequestURI(),
		Username: bindingURL.User.Username(),
		Password: password,
		URI:      bindingURL.String(),
		JDBCURI:  JDBCURI(bindingURL.Scheme, bindingURL.Hostname(), port[0], bindingURL.RequestURI(), bindingURL.User.Username(), password),
	}

	return binding, nil
}

func (b *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	b.Logger.Debug("unbind", lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
		detailsLogKey:    details,
	})

	return nil
}

func (b *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	b.Logger.Debug("update", lager.Data{
		instanceIDLogKey:   instanceID,
		detailsLogKey:      details,
		asyncAllowedLogKey: asyncAllowed,
	})

	spec := brokerapi.UpdateServiceSpec{
		IsAsync: true,
	}

	if !asyncAllowed {
		return spec, brokerapi.ErrAsyncRequired
	}

	service, err := b.ComposeCatalog.GetService(details.ServiceID)
	if err != nil {
		return spec, err
	}

	instanceName, err := makeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return spec, err
	}

	deployment, err := findDeployment(b.Compose, instanceName)
	if err != nil {
		return spec, err
	}

	if details.PlanID != details.PreviousValues.PlanID {
		return spec, fmt.Errorf("changing plans is not currently supported")
	}

	plan, err := service.GetPlan(details.PlanID)
	if err != nil {
		return spec, err
	}

	params := composeapi.ScalingsParams{
		DeploymentID: deployment.ID,
		Units:        plan.Metadata.Units,
	}

	recipe, errs := b.Compose.SetScalings(params)
	if len(errs) > 0 {
		return spec, squashErrors(errs)
	}

	operationData, err := makeOperationData("update", recipe.ID)
	if err != nil {
		return spec, err
	}

	spec.OperationData = operationData

	return spec, nil
}

func (b *Broker) LastOperation(context context.Context, instanceID, operationDataJson string) (brokerapi.LastOperation, error) {

	lastOperation := brokerapi.LastOperation{}
	operationData := OperationData{}
	err := json.Unmarshal([]byte(operationDataJson), &operationData)
	if err != nil {
		return lastOperation, err
	}

	b.Logger.Debug("last-operation", lager.Data{
		instanceIDLogKey:    instanceID,
		operationDataLogKey: operationData.RecipeID,
	})

	recipe, errs := b.Compose.GetRecipe(operationData.RecipeID)
	if len(errs) > 0 {
		return lastOperation, squashErrors(errs)
	}

	state := composeStatus2State[recipe.Status]

	if state == "" {
		state = brokerapi.Failed
	}

	lastOperation.State = state
	lastOperation.Description = recipe.StatusDetail

	return lastOperation, nil
}
