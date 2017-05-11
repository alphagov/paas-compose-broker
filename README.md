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
  cf set-env compose-broker BROKER_API_HOST 0.0.0.0
  cf set-env compose-broker LOG_LEVEL DEBUG
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
