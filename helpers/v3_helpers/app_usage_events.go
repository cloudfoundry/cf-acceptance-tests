package v3_helpers

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
)

type Entity struct {
	AppName       string `json:"app_name"`
	AppGuid       string `json:"app_guid"`
	State         string `json:"state"`
	BuildpackName string `json:"buildpack_name"`
	BuildpackGuid string `json:"buildpack_guid"`
	ParentAppName string `json:"parent_app_name"`
	ParentAppGuid string `json:"parent_app_guid"`
	ProcessType   string `json:"process_type"`
	TaskGuid      string `json:"task_guid"`
}
type AppUsageEvent struct {
	Entity `json:"entity"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
}

func UsageEventsInclude(events []AppUsageEvent, event AppUsageEvent) bool {
	found := false
	for _, e := range events {
		found = event.Entity.ParentAppName == e.Entity.ParentAppName &&
			event.Entity.ParentAppGuid == e.Entity.ParentAppGuid &&
			event.Entity.ProcessType == e.Entity.ProcessType &&
			event.Entity.State == e.Entity.State &&
			event.Entity.AppGuid == e.Entity.AppGuid &&
			event.Entity.TaskGuid == e.Entity.TaskGuid
		if found {
			break
		}
	}
	return found
}

func LastPageUsageEvents(TestSetup *workflowhelpers.ReproducibleTestSuiteSetup) []AppUsageEvent {
	var response AppUsageEvents

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		workflowhelpers.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response, Config.DefaultTimeoutDuration())
	})

	return response.Resources
}
