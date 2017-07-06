#!/bin/bash

export SKIP_COMPOSE_API_TESTS=true

ginkgo -r
