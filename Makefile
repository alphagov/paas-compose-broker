.PHONY: test unit integration

test: unit integration

unit:
	SKIP_COMPOSE_API_TESTS=true ginkgo -r

integration:
	ginkgo --nodes=2 --timeout 30m -r integration_tests
