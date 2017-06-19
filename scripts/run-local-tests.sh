#!/bin/bash

ACCOUNT_ID=XXX
ACCESS_TOKEN=XXX

export ACCOUNT_ID
export ACCESS_TOKEN

ginkgo -r -skipPackage=real_api
