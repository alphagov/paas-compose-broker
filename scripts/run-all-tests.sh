#!/bin/sh

if [ -z $COMPOSE_API_KEY ]; then
  COMPOSE_API_KEY=`PASSWORD_STORE_DIR=$HOME/.paas-pass pass compose/dev/access_token`
  export COMPOSE_API_KEY
fi

ginkgo --timeout 30m --nodes=8 -r "$@"
