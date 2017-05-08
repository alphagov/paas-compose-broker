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
	LogLevel      lager.LogLevel
	BrokerAPIHost string
	BrokerAPIPort string
	Username      string
	Password      string
}

func New() *Config {
	return &Config{}
}

func (c *Config) Get() error {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		return fmt.Errorf("Please export $LOG_LEVEL")
	}
	var ok bool
	c.LogLevel, ok = logLevels[strings.ToUpper(logLevel)]
	if !ok {
		fmt.Errorf("Invalid log level: ", logLevel)
	}

	c.BrokerAPIHost = os.Getenv("BROKER_API_HOST")
	if c.BrokerAPIHost == "" {
		return fmt.Errorf("Please export $BROKER_API_HOST")
	}

	c.BrokerAPIPort = os.Getenv("PORT")
	if c.BrokerAPIPort == "" {
		return fmt.Errorf("Please export $PORT")
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
