package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-compose-broker/broker"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"

	"testing"
)

var (
	brokerAPI http.Handler
	brokerUrl string
)

const (
	INSTANCE_CREATE_TIMEOUT = 4 * time.Second
	username                = "username"
	password                = "password"
	listenPort              = "8080"
)

func TestSuite(t *testing.T) {
	BeforeSuite(func() {

		os.Setenv("PORT", listenPort)
		os.Setenv("USERNAME", username)
		os.Setenv("PASSWORD", password)

		config, err := config.New()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		logger := lager.NewLogger("compose-broker")
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, config.LogLevel))

		catalog := catalog.New()
		catalogData := `
{
   "services":[
      {
      "id":"1",
         "name":"mongo",
         "description":"Compose MongoDB instance",
         "requires":[],
         "tags":[
            "mongo",
            "compose"
         ],
         "metadata":{
            "displayName":"mongo",
            "imageUrl":"https://webassets.mongodb.com/_com_assets/cms/MongoDB-Logo-5c3a7405a85675366beb3a5ec4c032348c390b3f142f5e6dddf1d78e2df5cb5c.png",
            "longDescription":"Compose MongoDB instance",
            "providerDisplayName":"GOV.UK PaaS",
            "documentationUrl":"https://compose.com/mongodb",
            "supportUrl":"https://www.cloud.service.gov.uk/support.html"
         },
         "plans":[
            {
               "id":"1",
               "name":"Mongo small",
               "description":"Small plan",
               "metadata":{
                  "bullets":[
                     "1 unit"
                  ],
                  "costs":[
                     {
                        "amount":{
                           "GBP":1
                        },
                        "unit":"MONTHLY"
                    }
                  ],
                  "displayName":"Mongo small"
	       }
	    }
         ]
      }
   ]
}
`
		err = catalog.Load([]byte(catalogData))
		if err != nil {
			logger.Error("catalog", err)
		}

		broker := broker.New(config, catalog, &logger)
		credentials := brokerapi.BrokerCredentials{
			Username: config.Username,
			Password: config.Password,
		}
		brokerAPI = brokerapi.New(broker, logger, credentials)

		brokerUrl = fmt.Sprintf("http://%s", broker.Config.ListenHost+":"+listenPort)
	})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}
