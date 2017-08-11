.PHONY: test unit integration

test: unit integration

unit:
	./scripts/run-local-tests.sh

integration:
	ginkgo -p --nodes=8 -r integration_tests
