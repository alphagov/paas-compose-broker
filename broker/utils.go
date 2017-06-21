package broker

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

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

func makeInstanceName(dbPrefix, instanceID string) (string, error) {
	if dbPrefix == "" {
		return "", errors.New("dbPrefix can't be empty")
	}
	if instanceID == "" {
		return "", errors.New("instanceID can't be empty")
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(dbPrefix), instanceID), nil
}
