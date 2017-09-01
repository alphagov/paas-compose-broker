.PHONY: test unit integration

test: unit integration

unit:
	SKIP_COMPOSE_API_TESTS=true ginkgo -r

integration:
	ginkgo -p --nodes=8 --timeout 30m -r integration_tests
