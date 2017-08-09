# PaaS Compose Service Broker

This is a work-in-progress implementation of a service broker for services provisioned via the [Compose API](https://apidocs.compose.com/). It is intended to be deployed as a CloudFoundry application.

## Deployment

* Use the example catalog:

  ```sh
  cp examples/catalog.json .
  ```

* Deploy to Cloud Foundry:

  ```sh
  cf push --no-start
  ```

* Set the required environment variables:

  ```sh
  cf set-env compose-broker USERNAME compose-broker
  cf set-env compose-broker PASSWORD unguessable
  ```

* Start the app:

  ```sh
  cf start compose-broker
  ```

* Check the catalog endpoint:

  ```sh
  curl -k -u compose-broker:unguessable https://compose-broker.${DEPLOY_ENV}.cloudfoundry-apps-domain.example.com/v2/catalog
  ```

* Register the service:

  ```
   cf update-service-broker compose-broker compose-broker "$COMPOSE_BROKER_PASS" "https://compose-broker.${APPS_DNS_ZONE_NAME}"
  ```
* Enable the service:

   ```
   cf enable-service-access mongodb
   ```

## Environmental variables

`USERNAME` - broker user name used for basic authentication
`PASSWORD` - username password
`DB_PREFIX` - a prefix that can be used to tag instances. Defaults to `compose-broker`
`CLUSTER_NAME` - a name of your enterprise cluster if you've got one and want to use it. Defaults to hosted compose
`COMPOSE_API_KEY` - your API key for Compose.


## Running tests

Prerequisites:

* [Ginkgo](https://onsi.github.io/ginkgo/) version `>= 1.4.0`
* An API key for [Compose](https://www.compose.com/) if running the full tests.

To run all tests (including integration tests):

```
COMPOSE_API_KEY=<key> ./scripts/run-all-tests.sh
```

To only run the unit tests you will need to use Docker for the database engine suite:

```
# Mongodb
docker run -p 27017:27017 --name mongo -e MONGO_INITDB_ROOT_PASSWORD= -d mongo:3.4

# Elasticsearch
# source: https://www.elastic.co/guide/en/elasticsearch/reference/current/docker.html
# Note: this must be run from the root directory of the repository
docker run -p 9200:9200 --name elasticsearch -v "${PWD}/dbengine/test/elasticsearch-docker-config.yml":/usr/share/elasticsearch/config/elasticsearch.yml -e http.host=0.0.0.0 -e transport.host=127.0.0.1 -d docker.elastic.co/elasticsearch/elasticsearch:5.5.1

# Check elasticsearch is running:
curl localhost:9200/_cluster/health

# Clean up afterwards
docker rm -f mongo elasticsearch
```

Then you can run them locally:

```
./scripts/run-local-tests.sh
```
