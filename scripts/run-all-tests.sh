#!/bin/bash

ACCOUNT_ID=$(PASSWORD_STORE_DIR=$HOME/.paas-pass  pass compose/account_id)
ACCESS_TOKEN=$(PASSWORD_STORE_DIR=$HOME/.paas-pass pass compose/dev/access_token)

export ACCOUNT_ID
export ACCESS_TOKEN

ginkgo -r
