package broker_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	"testing"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}

func MatchOperationJSON(expected string) types.GomegaMatcher {
	return &matchOperationJSONMatcher{
		expected: []byte(expected),
	}
}

type matchOperationJSONMatcher struct {
	expected []byte
}

func (matcher *matchOperationJSONMatcher) Match(actual interface{}) (bool, error) {
	var parsedBody struct {
		Operation string `json:"operation"`
	}
	actualBytes, ok := actual.([]byte)
	if !ok {
		return false, fmt.Errorf("Must pass a byte array, got: %s", actual)
	}
	err := json.Unmarshal(actualBytes, &parsedBody)
	if err != nil {
		return false, err
	}
	if parsedBody.Operation == "" {
		return false, fmt.Errorf("No operation data found in %s", actualBytes)
	}
	return MatchJSON(matcher.expected).Match(parsedBody.Operation)
}

func (matcher *matchOperationJSONMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected %s to have operation field %s", actual, matcher.expected)
}

func (matcher *matchOperationJSONMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("expected %s not to have operation field %s", actual, matcher.expected)
}
