#!/bin/bash

docker run -p 27017:27017 --name mongo -e MONGO_INITDB_ROOT_PASSWORD= -d mongo:3.4
docker run -p 9200:9200 --name elasticsearch -v "${PWD}/dbengine/test/elasticsearch-docker-config.yml":/usr/share/elasticsearch/config/elasticsearch.yml -e http.host=0.0.0.0 -e transport.host=127.0.0.1 -d docker.elastic.co/elasticsearch/elasticsearch:5.5.1

sleep 60s

SKIP_COMPOSE_API_TESTS=true ginkgo --timeout 30m -r

docker rm -f mongo elasticsearch