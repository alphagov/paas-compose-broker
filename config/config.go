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
	LogLevel   lager.LogLevel
	ListenPort string
	Username   string
	Password   string
}

func New() (*Config, error) {
	logLevel := lager.DEBUG
	logLevelFromEnv := os.Getenv("LOG_LEVEL")
	if logLevelFromEnv != "" {
		var ok bool
		logLevel, ok = logLevels[strings.ToUpper(logLevelFromEnv)]
		if !ok {
			return nil, fmt.Errorf("Invalid log level: ", logLevelFromEnv)
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
	return &Config{
		LogLevel:   logLevel,
		ListenPort: listenPort,
		Username:   username,
		Password:   password,
	}, nil
}
