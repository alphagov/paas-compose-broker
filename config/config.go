package config

import (
	"fmt"
	"os"
	"strings"

	"code.cloudfoundry.org/lager"
)

var (
	logLevels = map[string]lager.LogLevel{
		"DEBUG": lager.DEBUG,
		"INFO":  lager.INFO,
		"ERROR": lager.ERROR,
		"FATAL": lager.FATAL,
	}
)

type Config struct {
	APIToken    string
	LogLevel    lager.LogLevel
	ListenPort  string
	Username    string
	Password    string
	DBPrefix    string
	ClusterName string
}

func New() (*Config, error) {

	c := &Config{}

	c.LogLevel = lager.DEBUG
	logLevelFromEnv := os.Getenv("LOG_LEVEL")
	if logLevelFromEnv != "" {
		var ok bool
		c.LogLevel, ok = logLevels[strings.ToUpper(logLevelFromEnv)]
		if !ok {
			return nil, fmt.Errorf("Invalid log level: %s", logLevelFromEnv)
		}
	}

	c.ListenPort = os.Getenv("PORT")
	if c.ListenPort == "" {
		c.ListenPort = "8080"
	}

	c.Username = os.Getenv("USERNAME")
	if c.Username == "" {
		return nil, fmt.Errorf("Please export $USERNAME")
	}

	c.Password = os.Getenv("PASSWORD")
	if c.Password == "" {
		return nil, fmt.Errorf("Please export $PASSWORD")
	}

	c.DBPrefix = os.Getenv("DB_PREFIX")
	if c.DBPrefix == "" {
		c.DBPrefix = "compose-broker"
	}

	c.APIToken = os.Getenv("COMPOSE_API_KEY")
	if c.APIToken == "" {
		return nil, fmt.Errorf("Please export $COMPOSE_API_KEY")
	}

	c.ClusterName = os.Getenv("CLUSTER_NAME")

	return c, nil
}
