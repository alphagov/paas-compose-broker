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
	AccountID  string
	APIToken   string
	LogLevel   lager.LogLevel
	ListenPort string
	Username   string
	Password   string
	DBPrefix   string
}

func New() (*Config, error) {
	logLevel := lager.DEBUG
	logLevelFromEnv := os.Getenv("LOG_LEVEL")
	if logLevelFromEnv != "" {
		var ok bool
		logLevel, ok = logLevels[strings.ToUpper(logLevelFromEnv)]
		if !ok {
			return nil, fmt.Errorf("Invalid log level: %s", logLevelFromEnv)
		}
	}

	listenPort := "8080"
	listenPortFromEnv := os.Getenv("PORT")
	if listenPortFromEnv != "" {
		listenPort = listenPortFromEnv
	}

	username := os.Getenv("USERNAME")
	if username == "" {
		return nil, fmt.Errorf("Please export $USERNAME")
	}

	password := os.Getenv("PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("Please export $PASSWORD")
	}

	token := os.Getenv("ACCESS_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("Please export $ACCESS_TOKEN")
	}

	dbPrefix := os.Getenv("DB_PREFIX")
	if dbPrefix == "" {
		dbPrefix = "compose-broker"
	}

	return &Config{
		APIToken:   token,
		LogLevel:   logLevel,
		ListenPort: listenPort,
		Username:   username,
		Password:   password,
		DBPrefix:   dbPrefix,
	}, nil
}
