package matchers

import (
	"fmt"

	"github.com/cloudfoundry/noaa/events"
	"github.com/onsi/gomega/types"
)

type MetricsApp struct {
	AppGuid    string
	InstanceId int32
}

func NonZeroContainerMetricsFor(expected interface{}) types.GomegaMatcher {
	return &NonZeroContainerMetricsForMatcher{
		expected: expected,
	}
}

type NonZeroContainerMetricsForMatcher struct {
	expected interface{}
}

func (matcher *NonZeroContainerMetricsForMatcher) Match(actual interface{}) (success bool, err error) {
	envelope, ok := actual.(*events.Envelope)
	if !ok {
		return false, fmt.Errorf("actual is not of type *events.Envelope")
	}

	appInfo, ok := matcher.expected.(MetricsApp)
	if !ok {
		return false, fmt.Errorf("expected is not of type matchers.MetricsApp")
	}

	if cm := envelope.GetContainerMetric(); cm != nil {
		if cm.GetApplicationId() == appInfo.AppGuid && cm.GetInstanceIndex() == appInfo.InstanceId {
			if cm.GetMemoryBytes() > 0 && cm.GetDiskBytes() > 0 {
				return true, nil
			} else {
				return false, fmt.Errorf("expected non-zero container metrics for AppGuid: %s and InstanceId: %d", appInfo.AppGuid, appInfo.InstanceId)
			}
		}
	}

	return false, nil
}

func (matcher *NonZeroContainerMetricsForMatcher) FailureMessage(actual interface{}) (message string) {
	envelope, ok := actual.(*events.Envelope)
	if !ok {
		return "NonZeroContainerMetricsFor matcher: actual value must be an *events.Envelope"
	}
	appInfo, ok := matcher.expected.(MetricsApp)
	if !ok {
		return "NonZeroContainerMetricsFor matcher: expected is not of type matchers.MetricsApp"
	}

	return fmt.Sprintf(
		"Expected\n\t%#v\nto have application metrics for %s\n\t",
		envelope.GetContainerMetric(),
		appInfo.AppGuid,
	)
}

func (matcher *NonZeroContainerMetricsForMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	envelope, ok := actual.(*events.Envelope)
	if !ok {
		return "NonZeroContainerMetricsFor matcher: actual value must be an *events.Envelope"
	}
	appInfo, ok := matcher.expected.(MetricsApp)
	if !ok {
		return "NonZeroContainerMetricsFor matcher: expected is not of type matchers.MetricsApp"
	}

	return fmt.Sprintf(
		"Expected\n\t%#v\nnot to have application metrics for %s\n\t",
		envelope.GetContainerMetric(),
		appInfo.AppGuid,
	)
}
