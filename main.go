package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/compose"
	"github.com/alphagov/paas-compose-broker/config"
	"github.com/pivotal-cf/brokerapi"
)

var (
	catalogFilePath string
)

func main() {
	flag.StringVar(&catalogFilePath, "catalog", "./catalog.json", "Location of the catalog file")
	flag.Parse()
	config, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	logger := lager.NewLogger("compose-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, config.LogLevel))

	catalogFile, err := os.Open(catalogFilePath)
	if err != nil {
		logger.Error("opening catalog file", err)
		os.Exit(1)
	}
	newCatalog, err := catalog.Load(catalogFile)
	if err != nil {
		logger.Error("loading catalog", err)
		os.Exit(1)
	}
	if err := catalogFile.Close(); err != nil {
		logger.Error("closing catalog file", err)
		os.Exit(1)
	}

	composeapi, err := compose.NewClient(config.APIToken)
	if err != nil {
		logger.Error("could not create composeapi client", err)
		os.Exit(1)
	}

	brokerInstance, err := broker.New(composeapi, config, newCatalog, logger)
	if err != nil {
		logger.Error("could not initialise broker", err)
		os.Exit(1)
	}

	// The RequireTLS flag exists such that local tests can use insecure database
	// connections. In production it must always be set to true to forbid such
	// insecure connections.
	if !brokerInstance.RequireTLS {
		panic("The broker must be configured to refuse non-TLS connections.")
	}

	credentials := brokerapi.BrokerCredentials{
		Username: config.Username,
		Password: config.Password,
	}
	brokerAPI := brokerapi.New(brokerInstance, logger, credentials)

	http.Handle("/", brokerAPI)
	logger.Info("http-listen", lager.Data{"info": fmt.Sprintf("Service Broker started on " + "0.0.0.0:" + brokerInstance.Config.ListenPort)})
	logger.Error("http-listen", http.ListenAndServe(":"+brokerInstance.Config.ListenPort, nil))
}
