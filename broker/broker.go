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

type Credentials struct {
	Host                string `json:"host"`
	Port                string `json:"port"`
	Name                string `json:"name"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	URI                 string `json:"uri"`
	CACertificateBase64 string `json:"ca_certificate_base64"`
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

type Broker struct {
	Compose        compose.Client
	Config         *config.Config
	ComposeCatalog *catalog.ComposeCatalog
	Logger         lager.Logger
	AccountID      string
	ClusterID      string
}

func New(composeClient compose.Client, config *config.Config, catalog *catalog.ComposeCatalog, logger lager.Logger) (*Broker, error) {

	account, errs := composeClient.GetAccount()
	if len(errs) > 0 {
		return nil, fmt.Errorf("could not get account ID: %s", compose.SquashErrors(errs))
	}

	broker := Broker{
		Compose:        composeClient,
		Config:         config,
		ComposeCatalog: catalog,
		Logger:         logger,
		AccountID:      account.ID,
	}

	if config.ClusterName != "" {
		cluster, errs := composeClient.GetClusterByName(config.ClusterName)
		if len(errs) > 0 {
			return nil, fmt.Errorf("could not get cluster ID: %s", compose.SquashErrors(errs))
		}
		broker.ClusterID = cluster.ID
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
		AccountID:    b.AccountID,
		Datacenter:   ComposeDatacenter,
		DatabaseType: service.Name,
		Units:        plan.Metadata.Units,
		SSL:          true,
		ClusterID:    b.ClusterID,
	}

	deployment, errs := b.Compose.CreateDeployment(params)
	if len(errs) > 0 {
		return spec, compose.SquashErrors(errs)
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
	if err == errDeploymentNotFound {
		return spec, brokerapi.ErrInstanceDoesNotExist
	} else if err != nil {
		return spec, err
	}

	recipe, errs := b.Compose.DeprovisionDeployment(deployment.ID)
	if len(errs) > 0 {
		return spec, compose.SquashErrors(errs)
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
	if err == errDeploymentNotFound {
		return binding, brokerapi.ErrInstanceDoesNotExist
	} else if err != nil {
		return binding, err
	}

	deployment, errs := b.Compose.GetDeployment(deploymentMeta.ID)
	if len(errs) > 0 {
		return binding, compose.SquashErrors(errs)
	}
	if deployment.Connection.Direct == nil || len(deployment.Connection.Direct) < 1 {
		return binding, fmt.Errorf("failed to get connection string")
	}
	bindingURL, err := url.Parse(deployment.Connection.Direct[0])
	if err != nil {
		return binding, err
	}

	// FIXME: Follow up story should fix mongo connection string handling.
	// Right now we are hardcoding first host from the comma delimited list that Compose provides.
	// url.Parse() parses mongo connection string wrongly and doesn't return an error
	// so url.Port() returns port like "18899,aws-eu-west-1-portal.7.dblayer.com:18899"
	port := strings.Split(bindingURL.Port(), ",")

	username := bindingURL.User.Username()
	password, _ := bindingURL.User.Password()
	dbName := strings.TrimPrefix(bindingURL.Path, "/")

	binding.Credentials = Credentials{
		Host:                bindingURL.Hostname(),
		Port:                port[0],
		Name:                dbName,
		Username:            username,
		Password:            password,
		URI:                 bindingURL.String(),
		CACertificateBase64: deployment.CACertificateBase64,
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
	if err == errDeploymentNotFound {
		return spec, brokerapi.ErrInstanceDoesNotExist
	} else if err != nil {
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
		return spec, compose.SquashErrors(errs)
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
		return lastOperation, compose.SquashErrors(errs)
	}

	state := composeStatus2State[recipe.Status]

	if state == "" {
		state = brokerapi.Failed
	}

	lastOperation.State = state
	lastOperation.Description = recipe.StatusDetail

	return lastOperation, nil
}
