package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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
	config := config.New()
	err := config.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	logger := lager.NewLogger("compose-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, config.LogLevel))

	catalogData, err := ioutil.ReadFile(catalogFilePath)
	if err != nil {
		logger.Error("catalog", err)
		os.Exit(1)
	}
	catalog := catalog.New()
	err = catalog.Load(catalogData)
	if err != nil {
		logger.Error("catalog", err)
		os.Exit(1)
	}

	broker := broker.New(config, catalog, &logger)
	credentials := brokerapi.BrokerCredentials{
		Username: config.Username,
		Password: config.Password,
	}
	brokerAPI := brokerapi.New(broker, logger, credentials)

	http.Handle("/", brokerAPI)
	logger.Info("http-listen", lager.Data{"info": fmt.Sprintf("Service Broker started on " + broker.Config.BrokerAPIHost + ":" + broker.Config.BrokerAPIPort)})
	logger.Error("http-listen", http.ListenAndServe(broker.Config.BrokerAPIHost+":"+broker.Config.BrokerAPIPort, nil))
}
