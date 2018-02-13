.PHONY: test unit integration

test: unit integration

unit: compose/fakes/fake_client.go
	SKIP_COMPOSE_API_TESTS=true ginkgo -r

integration:
	ginkgo --nodes=2 --timeout 60m -r integration_tests

compose/fakes/fake_client.go: compose/*.go
	go generate ./compose
