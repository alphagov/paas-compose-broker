package broker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alphagov/paas-compose-broker/compose"
	composeapi "github.com/compose/gocomposeapi"
)

func findDeployment(compose compose.Client, name string) (*composeapi.Deployment, error) {
	deployments, errs := compose.GetDeployments()
	if len(errs) > 0 {
		return nil, squashErrors(errs)
	}

	for _, deployment := range *deployments {
		if deployment.Name == name {
			return &deployment, nil
		}
	}

	return nil, fmt.Errorf("deployment: not found")
}

func squashErrors(errs []error) error {
	var s []string

	for _, err := range errs {
		s = append(s, err.Error())
	}

	return fmt.Errorf("%s", strings.Join(s, "; "))
}

func JDBCURI(scheme, hostname, port, dbname, username, password string) string {
	return fmt.Sprintf("jdbc:mongodb://%s:%s/%s?user=%s&password=%s", hostname, port, dbname, username, password)
}

func makeOperationData(operationType, recipeID string) (string, error) {

	operationData := OperationData{
		Type:     operationType,
		RecipeID: recipeID,
	}

	data, err := json.Marshal(operationData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
