#!/bin/bash

ACCOUNT_ID=XXX
ACCESS_TOKEN=XXX

export ACCOUNT_ID
export ACCESS_TOKEN

# shellcheck disable=SC2046
go test -v $(go list ./... | grep -v '/vendor/' | grep -v '/integration_tests')
ginkgo -focus='Broker with fake Compose client' integration_tests/
