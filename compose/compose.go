package compose

import (
	composeapi "github.com/compose/gocomposeapi"
)

type Client interface {
	CreateDeployment(composeapi.DeploymentParams) (*composeapi.Deployment, []error)
	DeprovisionDeployment(string) (*composeapi.Recipe, []error)
	GetDeployment(string) (*composeapi.Deployment, []error)
	GetDeployments() (*[]composeapi.Deployment, []error)
	GetRecipe(string) (*composeapi.Recipe, []error)
	SetScalings(composeapi.ScalingsParams) (*composeapi.Recipe, []error)
}

func NewClient(apiToken string) (*composeapi.Client, error) {
	return composeapi.NewClient(apiToken)
}
