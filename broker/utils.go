package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alphagov/paas-compose-broker/compose"
	composeapi "github.com/compose/gocomposeapi"
)

func findDeployment(c compose.Client, name string) (*composeapi.Deployment, error) {
	deployments, errs := c.GetDeployments()
	if len(errs) > 0 {
		return nil, compose.SquashErrors(errs)
	}

	for _, deployment := range *deployments {
		if deployment.Name == name {
			return &deployment, nil
		}
	}

	return nil, fmt.Errorf("deployment: not found")
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

func makeInstanceName(dbPrefix, instanceID string) (string, error) {
	if dbPrefix == "" {
		return "", errors.New("dbPrefix can't be empty")
	}
	if instanceID == "" {
		return "", errors.New("instanceID can't be empty")
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(dbPrefix), instanceID), nil
}
