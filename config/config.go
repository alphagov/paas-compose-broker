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
	ListenHost string
	ListenPort string
	Username   string
	Password   string
}

func New() *Config {
	return &Config{}
}

func (c *Config) Get() error {
	c.LogLevel = lager.DEBUG
	logLevelFromEnv := os.Getenv("LOG_LEVEL")
	if logLevelFromEnv != "" {
		var ok bool
		c.LogLevel, ok = logLevels[strings.ToUpper(logLevelFromEnv)]
		if !ok {
			return fmt.Errorf("Invalid log level: ", logLevelFromEnv)
		}
	}

	c.ListenHost = "0.0.0.0"
	listenHostFromEnv := os.Getenv("LISTEN_HOST")
	if listenHostFromEnv != "" {
		c.ListenHost = listenHostFromEnv
	}

	c.ListenPort = "8080"
	listenPortFromEnv := os.Getenv("PORT")
	if listenPortFromEnv != "" {
		c.ListenPort = listenPortFromEnv
	}

	c.Username = os.Getenv("USERNAME")
	if c.Username == "" {
		return fmt.Errorf("Please export $USERNAME")
	}

	c.Password = os.Getenv("PASSWORD")
	if c.Password == "" {
		return fmt.Errorf("Please export $PASSWORD")
	}
	return nil
}
