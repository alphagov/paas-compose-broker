#!/bin/bash

ACCESS_TOKEN=$(PASSWORD_STORE_DIR=$HOME/.paas-pass pass compose/dev/access_token)

export ACCESS_TOKEN

ginkgo -r
