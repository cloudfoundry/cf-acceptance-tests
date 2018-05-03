package honeycomb

import (
	"strings"

	"github.com/cloudfoundry/custom-cats-reporters/honeycomb/client"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
)

type SpecEvent struct {
	Description string
	State       string
}

type honeyCombReporter struct {
	client     client.Client
	globalTags map[string]interface{}
	customTags map[string]interface{}
}

func New(client client.Client) honeyCombReporter {
	return honeyCombReporter{client: client}
}

func (hr honeyCombReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	specEvent := SpecEvent{
		State:       getTestState(specSummary.State),
		Description: createTestDescription(specSummary.ComponentTexts),
	}

	hr.client.SendEvent(specEvent, hr.globalTags, hr.customTags)
}

func (hr honeyCombReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	hr.client.SendEvent(*summary, hr.globalTags, hr.customTags)
}

func (hr honeyCombReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
}
func (hr honeyCombReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {}
func (hr honeyCombReporter) SpecWillRun(specSummary *types.SpecSummary)         {}
func (hr honeyCombReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary)  {}

func (hr *honeyCombReporter) SetGlobalTags(globalTags map[string]interface{}) {
	hr.globalTags = globalTags
}

func (hr *honeyCombReporter) SetCustomTags(customTags map[string]interface{}) {
	hr.customTags = customTags
}

func getTestState(state types.SpecState) string {
	switch state {
	case types.SpecStatePassed:
		return "passed"
	case types.SpecStateFailed:
		return "failed"
	case types.SpecStatePending:
		return "pending"
	case types.SpecStateSkipped:
		return "skipped"
	case types.SpecStatePanicked:
		return "panicked"
	case types.SpecStateTimedOut:
		return "timedOut"
	case types.SpecStateInvalid:
		return "invalid"
	default:
		panic("unknown spec state")
	}
}

func createTestDescription(componentTexts []string) string {
	return strings.Join(componentTexts, " | ")
}
