package compose

import (
	"fmt"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

type Client interface {
	GetAccount() (*composeapi.Account, []error)
	GetClusters() (*[]composeapi.Cluster, []error)
	GetCluster(string) (*composeapi.Cluster, []error)
	GetClusterByName(string) (*composeapi.Cluster, []error)
	CreateDeployment(composeapi.DeploymentParams) (*composeapi.Deployment, []error)
	DeprovisionDeployment(string) (*composeapi.Recipe, []error)
	GetDeployment(string) (*composeapi.Deployment, []error)
	GetDeploymentByName(string) (*composeapi.Deployment, []error)
	GetDeployments() (*[]composeapi.Deployment, []error)
	GetRecipe(string) (*composeapi.Recipe, []error)
	SetScalings(composeapi.ScalingsParams) (*composeapi.Recipe, []error)
}

func NewClient(apiToken string) (Client, error) {
	return composeapi.NewClient(apiToken)
}

func SquashErrors(errs []error) error {
	var s []string

	for _, err := range errs {
		s = append(s, err.Error())
	}

	return fmt.Errorf("%s", strings.Join(s, "; "))
}
