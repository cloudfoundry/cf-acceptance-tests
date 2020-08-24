package app_helpers

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
)

type AppUsageEvent struct {
	Guid      string `json:"guid"`
	Buildpack struct {
		Name string `json:"name"`
		Guid string `json:"guid"`
	} `json:"buildpack"`
	Task struct {
		Guid string `json:"guid"`
	} `json:"task"`
	State struct {
		Current string `json:"current"`
	} `json:"state"`
	App struct {
		Name string `json:"name"`
		Guid string `json:"guid"`
	} `json:"app"`
	Process struct {
		Type string `json:"type"`
		Guid string `json:"guid"`
	} `json:"process"`
}

type Pagination struct {
	Next struct {
		href string `json:"href"`
	} `json:"next"`
}

type AppUsageEvents struct {
	Resources  []AppUsageEvent `struct:"resources"`
	Pagination Pagination      `json:"pagination"`
}

func UsageEventsInclude(events []AppUsageEvent, event AppUsageEvent) bool {
	found := false
	for _, e := range events {
		found = event.App.Name == e.App.Name &&
			event.App.Guid == e.App.Guid &&
			event.Process.Type == e.Process.Type &&
			event.State.Current == e.State.Current &&
			event.Process.Guid == e.Process.Guid &&
			event.Task.Guid == e.Task.Guid
		if found {
			break
		}
	}
	return found
}

func LastAppUsageEventGuid(testSetup *workflowhelpers.ReproducibleTestSuiteSetup) string {
	var response AppUsageEvents

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		workflowhelpers.ApiRequest("GET", "/v3/app_usage_events?per_page=1&order_by=-created_at&page=1", &response, Config.DefaultTimeoutDuration())
	})

	return response.Resources[0].Guid
}

// Returns all app usage events that occured since the given app usage event guid
func UsageEventsAfterGuid(guid string) []AppUsageEvent {
	resources := make([]AppUsageEvent, 0)

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		firstPageUrl := "/v3/app_usage_events?per_page=150&order_by=-created_at&page=1&after_guid=" + guid
		url := firstPageUrl

		for {
			var response AppUsageEvents
			workflowhelpers.ApiRequest("GET", url, &response, Config.DefaultTimeoutDuration())

			resources = append(resources, response.Resources...)

			if len(response.Resources) == 0 || response.Pagination.Next.href == "" {
				break
			}

			url = response.Pagination.Next.href
		}
	})

	return resources
}
