package realapi_test

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-compose-broker/catalog"
	"github.com/alphagov/paas-compose-broker/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	newConfig  *config.Config
	newCatalog *catalog.ComposeCatalog
	logger     lager.Logger
	brokerUrl  string
	username   string
	password   string
	dbprefix   string
	err        error
)

const (
	INSTANCE_CREATE_TIMEOUT = 15 * time.Minute
	randLength              = 10
	letters                 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func TestSuite(t *testing.T) {
	BeforeSuite(func() {

		username = randString(randLength)
		password = randString(randLength)
		dbprefix = "test-suite"

		os.Setenv("USERNAME", username)
		os.Setenv("PASSWORD", password)
		os.Setenv("DB_PREFIX", dbprefix)

		newConfig, err = config.New()
		Expect(err).ToNot(HaveOccurred())

		logger = lager.NewLogger("compose-broker")
		logger.RegisterSink(lager.NewWriterSink(os.Stdout, newConfig.LogLevel))

		catalogData := strings.NewReader(`
{
  "services": [{
    "id": "36f8bf47-c9e7-46d9-880f-5dfc838d05cb",
    "name": "mongodb",
    "description": "Compose MongoDB instance",
    "requires": [],
    "tags": [
      "mongo",
      "compose"
    ],
    "metadata": {
      "displayName": "MongoDB",
      "imageUrl": "https://webassets.mongodb.com/_com_assets/cms/MongoDB-Logo-5c3a7405a85675366beb3a5ec4c032348c390b3f142f5e6dddf1d78e2df5cb5c.png",
      "longDescription": "Compose MongoDB instance",
      "providerDisplayName": "GOV.UK PaaS",
      "documentationUrl": "https://compose.com/mongodb",
      "supportUrl": "https://www.cloud.service.gov.uk/support.html"
    },
    "plans": [{
      "id": "fdfd4fc1-ce69-451c-a436-c2e2795b9abe",
      "name": "small",
      "description": "1GB Storage / 102MB RAM at $35.00/month.",
      "metadata": {
        "displayName": "Mongo Small",
        "bullets": [],
        "units": 1,
        "costs": [{
          "amount": {
            "USD": 35
          },
          "unit": "MONTHLY"
        }]
      }
    }]
  }]
}
`)

		newCatalog, err = catalog.Load(catalogData)
		Expect(err).ToNot(HaveOccurred())
	})
	RegisterFailHandler(Fail)
	RunSpecs(t, "Real API Suite")
}

func randString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
