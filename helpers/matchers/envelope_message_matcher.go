package matchers

import (
	"github.com/cloudfoundry/noaa/events"
	"github.com/onsi/gomega/types"

	"fmt"
	"strings"
)

func EnvelopeContainingMessageLike(expected interface{}) types.GomegaMatcher {
	return &EnvelopeContainingMessageLikeMatcher{
		expected: expected,
	}
}

type EnvelopeContainingMessageLikeMatcher struct {
	expected interface{}
}

func (matcher *EnvelopeContainingMessageLikeMatcher) Match(actual interface{}) (success bool, err error) {
	envelope, ok := actual.(*events.Envelope)
	if !ok {
		return false, fmt.Errorf("EnvelopeContainingMessageLikeMatcher matcher: actual value must be an events.Envelope")
	}

	expectedMessage, ok := matcher.expected.(string)
	if !ok {
		return false, fmt.Errorf("EnvelopeContainingMessageLikeMatcher matcher: expected value must be a string")
	}

	return strings.Contains(string(envelope.GetLogMessage().GetMessage()), expectedMessage), nil
}

func (matcher *EnvelopeContainingMessageLikeMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto have log message containing\n\t%#v", actual, matcher.expected)
}

func (matcher *EnvelopeContainingMessageLikeMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to have log message containing\n\t%#v", actual, matcher.expected)
}
