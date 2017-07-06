package fakes

import (
	"fmt"

	composeapi "github.com/compose/gocomposeapi"
)

type FakeComposeClient struct {
	Deployments             []composeapi.Deployment
	CreateDeploymentParams  composeapi.DeploymentParams
	DeprovisionDeploymentID string
	SetScalingsParams       composeapi.ScalingsParams
	GetRecipeID             string
	GetRecipeErr            error
	GetRecipeStatus         string
}

func New() *FakeComposeClient {
	return &FakeComposeClient{}
}

func (fcc *FakeComposeClient) CreateDeployment(deploymentParams composeapi.DeploymentParams) (*composeapi.Deployment, []error) {
	fcc.CreateDeploymentParams = deploymentParams
	return &composeapi.Deployment{ProvisionRecipeID: "provision-recipe-id"}, []error{}
}

func (fcc *FakeComposeClient) DeprovisionDeployment(deploymentID string) (*composeapi.Recipe, []error) {
	fcc.DeprovisionDeploymentID = deploymentID
	return &composeapi.Recipe{ID: "deprovision-recipe-id"}, []error{}
}

func (fcc *FakeComposeClient) GetDeployment(deploymentID string) (*composeapi.Deployment, []error) {
	for _, deployment := range fcc.Deployments {
		if deployment.ID == deploymentID {
			return &deployment, nil
		}
	}

	return nil, []error{fmt.Errorf("deployment: not found")}
}

func (fcc *FakeComposeClient) GetDeployments() (*[]composeapi.Deployment, []error) {
	return &fcc.Deployments, []error{}
}

func (fcc *FakeComposeClient) GetRecipe(recipeID string) (*composeapi.Recipe, []error) {
	fcc.GetRecipeID = recipeID
	if fcc.GetRecipeErr != nil {
		return nil, []error{fcc.GetRecipeErr}
	}
	return &composeapi.Recipe{Status: fcc.GetRecipeStatus}, nil
}

func (fcc *FakeComposeClient) SetScalings(scalingParams composeapi.ScalingsParams) (*composeapi.Recipe, []error) {
	fcc.SetScalingsParams = scalingParams
	return &composeapi.Recipe{}, []error{}
}
