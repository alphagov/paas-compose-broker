package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/alphagov/paas-compose-broker/compose"
	composeapi "github.com/compose/gocomposeapi"
)

var errDeploymentNotFound = errors.New("Deployment not found")

func findDeployment(c compose.Client, name string) (*composeapi.Deployment, error) {
	deployment, errs := c.GetDeploymentByName(name)
	if len(errs) > 0 {
		if strings.Contains(errs[0].Error(), "deployment not found") {
			return nil, errDeploymentNotFound
		}
		return nil, compose.SquashErrors(errs)
	}
	if deployment == nil {
		return nil, errDeploymentNotFound
	}
	return deployment, nil
}

func makeOperationData(operationType, recipeID string, whitelistRecipeIDs []string) (string, error) {

	operationData := OperationData{
		Type:               operationType,
		RecipeID:           recipeID,
		WhitelistRecipeIDs: whitelistRecipeIDs,
	}

	data, err := json.Marshal(operationData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func MakeInstanceName(dbPrefix, instanceID string) (string, error) {
	if dbPrefix == "" {
		return "", errors.New("dbPrefix can't be empty")
	}
	if instanceID == "" {
		return "", errors.New("instanceID can't be empty")
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(dbPrefix), instanceID), nil
}

func newestRestorableBackup(backups []composeapi.Backup) *composeapi.Backup {
	var newest *composeapi.Backup
	for i, backup := range backups {
		if !backup.IsRestorable {
			continue
		}
		if newest == nil || backup.CreatedAt.After(newest.CreatedAt) {
			newest = &backups[i]
		}
	}
	return newest
}
