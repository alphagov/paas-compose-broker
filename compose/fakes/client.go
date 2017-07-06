package fakes

import (
	"fmt"

	composeapi "github.com/compose/gocomposeapi"
)

type FakeComposeClient struct {
	// Error to be returned from any call if set.
	GlobalError             error
	Account                 composeapi.Account
	Clusters                []composeapi.Cluster
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

func (fcc *FakeComposeClient) GetAccount() (*composeapi.Account, []error) {
	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}
	return &fcc.Account, nil
}

func (fcc *FakeComposeClient) GetClusters() (*[]composeapi.Cluster, []error) {
	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}
	return &fcc.Clusters, nil
}

func (fcc *FakeComposeClient) GetClusterByName(clusterName string) (*composeapi.Cluster, []error) {
	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	for _, c := range fcc.Clusters {
		if c.Name == clusterName {
			return &c, nil
		}
	}

	return nil, []error{fmt.Errorf("cluster: not found")}
}

func (fcc *FakeComposeClient) CreateDeployment(deploymentParams composeapi.DeploymentParams) (*composeapi.Deployment, []error) {
	fcc.CreateDeploymentParams = deploymentParams

	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	return &composeapi.Deployment{ProvisionRecipeID: "provision-recipe-id"}, []error{}
}

func (fcc *FakeComposeClient) DeprovisionDeployment(deploymentID string) (*composeapi.Recipe, []error) {
	fcc.DeprovisionDeploymentID = deploymentID

	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	return &composeapi.Recipe{ID: "deprovision-recipe-id"}, []error{}
}

func (fcc *FakeComposeClient) GetDeployment(deploymentID string) (*composeapi.Deployment, []error) {
	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	for _, deployment := range fcc.Deployments {
		if deployment.ID == deploymentID {
			return &deployment, nil
		}
	}

	return nil, []error{fmt.Errorf("deployment: not found")}
}

func (fcc *FakeComposeClient) GetDeployments() (*[]composeapi.Deployment, []error) {
	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	return &fcc.Deployments, []error{}
}

func (fcc *FakeComposeClient) GetRecipe(recipeID string) (*composeapi.Recipe, []error) {
	fcc.GetRecipeID = recipeID

	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	if fcc.GetRecipeErr != nil {
		return nil, []error{fcc.GetRecipeErr}
	}
	return &composeapi.Recipe{Status: fcc.GetRecipeStatus}, nil
}

func (fcc *FakeComposeClient) SetScalings(scalingParams composeapi.ScalingsParams) (*composeapi.Recipe, []error) {
	fcc.SetScalingsParams = scalingParams

	if fcc.GlobalError != nil {
		return nil, []error{fcc.GlobalError}
	}

	return &composeapi.Recipe{}, []error{}
}
