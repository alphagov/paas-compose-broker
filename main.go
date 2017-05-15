package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
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
	newCatalog, err := catalog.New(catalogFile)
	if err != nil {
		logger.Error("loading catalog", err)
		os.Exit(1)
	}
	if err := catalogFile.Close(); err != nil {
		logger.Error("closing catalog file", err)
		os.Exit(1)
	}

	broker := broker.New(config, newCatalog, &logger)
	credentials := brokerapi.BrokerCredentials{
		Username: config.Username,
		Password: config.Password,
	}
	brokerAPI := brokerapi.New(broker, logger, credentials)

	http.Handle("/", brokerAPI)
	logger.Info("http-listen", lager.Data{"info": fmt.Sprintf("Service Broker started on " + "0.0.0.0:" + broker.Config.ListenPort)})
	logger.Error("http-listen", http.ListenAndServe(":"+broker.Config.ListenPort, nil))
}
