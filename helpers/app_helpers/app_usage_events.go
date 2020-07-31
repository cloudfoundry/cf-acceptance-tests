package app_helpers

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
)

//type Entity struct {
//	AppName       string `json:"app_name"`
//	AppGuid       string `json:"app_guid"`
//	State         string `json:"state"`
//	BuildpackName string `json:"buildpack_name"`
//	BuildpackGuid string `json:"buildpack_guid"`
//	ParentAppName string `json:"parent_app_name"`
//	ParentAppGuid string `json:"parent_app_guid"`
//	ProcessType   string `json:"process_type"`
//	TaskGuid      string `json:"task_guid"`
//}

type Metadata struct {
	Guid string `json:"guid"`
}

type UsageState struct {
	Current  string `json:"current"`
	Previous string `json:"previous"`
}

type UsageApp struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

type UsageProcess struct {
	Guid string `json:"guid"`
	Type string `json:"type"`
}

type UsageSpace struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

type UsageOrganization struct {
	Guid string `json:"guid"`
}

type UsageBuildpack struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

type UsageTask struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

type UsageMemory struct {
	Current  json.Number `json:"current"`
	Previous json.Number `json:"previous"`
}

type UsageInstanceCount struct {
	Current  json.Number `json:"current"`
	Previous json.Number `json:"previous"`
}

type AppUsageEvent struct {
	Guid          string        `json:"guid"`
	State         UsageState         `json:"state"`
	App           UsageApp           `json:"app"`
	Process       UsageProcess       `json:"process"`
	Space         UsageSpace         `json:"space"`
	Organization  UsageOrganization  `json:"organization"`
	Buildpack     UsageBuildpack     `json:"buildpack"`
	Task          UsageTask          `json:"task"`
	Memory        UsageMemory        `json:"memory_in_mb_per_instance"`
	InstanceCount UsageInstanceCount `json:"instance_count"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
	NextUrl   string          `json:"next_url"`
}

func UsageEventsInclude(events []AppUsageEvent, event AppUsageEvent) bool {
	found := false
	//fmt.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	//fmt.Printf("looking for: Event{app_guid: %s, process_type: %s, state: %s, process_guid: %s, task_guid: %s} \n", event.App.Guid, event.Process.Type, event.State.Current, event.Process.Guid, event.Task.Guid)
	for _, e := range events {
		//fmt.Printf("comparing to: Event{app_guid: %s, process_type: %s, state: %s, process_guid: %s, task_guid: %s} ..... TRIVIA this event has memory alloted: %s\n", e.App.Guid, e.Process.Type, e.State.Current, e.Process.Guid, e.Task.Guid, e.Memory.Current)
		found = event.App.Guid == e.App.Guid &&
			event.Process.Type == e.Process.Type &&
			event.State.Current == e.State.Current &&
			//event.State.Previous == e.State.Previous && <-- should we match on this?
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

			if len(response.Resources) == 0 || response.NextUrl == "" {
				break
			}

			url = response.NextUrl
		}
	})

	return resources
}
