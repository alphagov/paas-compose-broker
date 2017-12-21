package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose"
	"github.com/alphagov/paas-compose-broker/config"
	"github.com/alphagov/paas-compose-broker/dbengine"
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

type OperationData struct {
	Type               string   `json:"type"`
	RecipeID           string   `json:"recipe_id"`
	WhitelistRecipeIDs []string `json:"whitelist_recipe_ids"`
}

func lookupBrokerAPIState(composeStatus string) brokerapi.LastOperationState {
	state, ok := composeStatus2BrokerAPIState[composeStatus]
	if !ok {
		return brokerapi.Failed
	}
	return state
}

var composeStatus2BrokerAPIState = map[string]brokerapi.LastOperationState{
	"complete": brokerapi.Succeeded,
	"running":  brokerapi.InProgress,
	"waiting":  brokerapi.InProgress,
	"failed":   brokerapi.Failed,
}

type Broker struct {
	Compose          compose.Client
	Config           *config.Config
	Catalog          *catalog.Catalog
	Logger           lager.Logger
	AccountID        string
	ClusterID        string
	DBEngineProvider dbengine.Provider
}

func New(composeClient compose.Client, dbEngineProvider dbengine.Provider, config *config.Config, catalog *catalog.Catalog, logger lager.Logger) (*Broker, error) {

	account, errs := composeClient.GetAccount()
	if len(errs) > 0 {
		return nil, fmt.Errorf("could not get account ID: %s", compose.SquashErrors(errs))
	}

	broker := Broker{
		Compose:          composeClient,
		Config:           config,
		Catalog:          catalog,
		Logger:           logger,
		AccountID:        account.ID,
		DBEngineProvider: dbEngineProvider,
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
	services := []brokerapi.Service{}
	for _, s := range b.Catalog.Services {
		services = append(services, s.Service)
	}
	return services
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

	service, err := b.Catalog.GetService(details.ServiceID)
	if err != nil {
		return spec, err
	}

	plan, err := service.GetPlan(details.PlanID)
	if err != nil {
		return spec, err
	}

	instanceName, err := MakeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return spec, err
	}

	params := composeapi.DeploymentParams{
		Name:         instanceName,
		AccountID:    b.AccountID,
		Datacenter:   ComposeDatacenter,
		DatabaseType: plan.Compose.DatabaseType,
		Units:        plan.Compose.Units,
		SSL:          true,
		ClusterID:    b.ClusterID,
	}

	deployment, errs := b.Compose.CreateDeployment(params)
	if len(errs) > 0 {
		return spec, compose.SquashErrors(errs)
	}

	whitelistRecipeIDs := []string{}
	for _, ip := range b.Config.IPWhitelist {
		whitelistParams := composeapi.DeploymentWhitelistParams{
			IP:          ip,
			Description: fmt.Sprintf("Allow %s to access deployment", ip),
		}
		whitelistRecipe, whitelistErrs := b.Compose.CreateDeploymentWhitelist(deployment.ID, whitelistParams)
		if len(whitelistErrs) > 0 {
			return spec, compose.SquashErrors(whitelistErrs)
		}
		if whitelistRecipe == nil {
			return spec, errors.New("malformed response from Compose: no pending whitelist recipe received")
		}
		if whitelistRecipe.ID == "" {
			return spec, errors.New("malformed response from Compose: invalid whitelist recipe ID")
		}
		whitelistRecipeIDs = append(whitelistRecipeIDs, whitelistRecipe.ID)
	}

	operationData, err := makeOperationData("provision", deployment.ProvisionRecipeID, whitelistRecipeIDs)
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

	instanceName, err := MakeInstanceName(b.Config.DBPrefix, instanceID)
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

	operationData, err := makeOperationData("deprovision", recipe.ID, []string{})
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

	instanceName, err := MakeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return binding, err
	}

	deployment, err := findDeployment(b.Compose, instanceName)
	if err == errDeploymentNotFound {
		return binding, brokerapi.ErrInstanceDoesNotExist
	} else if err != nil {
		return binding, err
	}

	if deployment.Connection.Direct == nil || len(deployment.Connection.Direct) < 1 {
		return binding, fmt.Errorf("failed to get connection string")
	}

	dbEngine, err := b.DBEngineProvider.GetDBEngine(deployment)
	if err != nil {
		return binding, err
	}

	binding.Credentials, err = dbEngine.GenerateCredentials(instanceID, bindingID)
	return binding, err
}

func (b *Broker) Unbind(context context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	b.Logger.Debug("unbind", lager.Data{
		instanceIDLogKey: instanceID,
		bindingIDLogKey:  bindingID,
		detailsLogKey:    details,
	})

	instanceName, err := MakeInstanceName(b.Config.DBPrefix, instanceID)
	if err != nil {
		return err
	}

	deployment, err := findDeployment(b.Compose, instanceName)
	if err == errDeploymentNotFound {
		return brokerapi.ErrInstanceDoesNotExist
	} else if err != nil {
		return err
	}

	dbEngine, err := b.DBEngineProvider.GetDBEngine(deployment)
	if err != nil {
		return err
	}

	return dbEngine.RevokeCredentials(instanceID, bindingID)
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

	service, err := b.Catalog.GetService(details.ServiceID)
	if err != nil {
		return spec, err
	}

	instanceName, err := MakeInstanceName(b.Config.DBPrefix, instanceID)
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
		Units:        plan.Compose.Units,
	}

	recipe, errs := b.Compose.SetScalings(params)
	if len(errs) > 0 {
		return spec, compose.SquashErrors(errs)
	}

	operationData, err := makeOperationData("update", recipe.ID, []string{})
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

	deploymentRecipe, errs := b.Compose.GetRecipe(operationData.RecipeID)
	if len(errs) > 0 {
		return lastOperation, compose.SquashErrors(errs)
	}
	deploymentState := lookupBrokerAPIState(deploymentRecipe.Status)
	if deploymentState != brokerapi.Succeeded {
		return brokerapi.LastOperation{
			State:       deploymentState,
			Description: deploymentRecipe.StatusDetail,
		}, nil
	}

	for _, recipeID := range operationData.WhitelistRecipeIDs {
		whitelistRecipe, errs := b.Compose.GetRecipe(recipeID)
		if len(errs) > 0 {
			return lastOperation, compose.SquashErrors(errs)
		}
		state := lookupBrokerAPIState(whitelistRecipe.Status)

		if state != brokerapi.Succeeded {
			return brokerapi.LastOperation{
				State:       state,
				Description: whitelistRecipe.StatusDetail,
			}, nil
		}
	}

	return brokerapi.LastOperation{
		State:       deploymentState,
		Description: deploymentRecipe.StatusDetail,
	}, nil
}
