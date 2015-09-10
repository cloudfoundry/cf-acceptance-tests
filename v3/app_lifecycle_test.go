package v3

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("v3 app lifecycle", func() {
	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo"=>"bar"`
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token := GetAuthToken()
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
	})

	It("can run apps", func() {
		dropletGuid := StagePackage(packageGuid, "{}")
		WaitForDropletToStage(dropletGuid)

		AssignDropletToApp(appGuid, dropletGuid)

		var webProcess Process
		//var workerProcess Process
		processes := getProcess(appGuid, appName)
		for _, process := range processes {
			if process.Type == "web" {
				webProcess = process
			} else if process.Type == "worker" {
				//	workerProcess = process
			}
		}

		Expect(webProcess.Guid).ToNot(BeEmpty())
		//Expect(workerProcess.Guid).ToNot(BeEmpty())

		CreateAndMapRoute(appGuid, context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, webProcess.Name)

		StartApp(appGuid)

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

		output := helpers.CurlApp(webProcess.Name, "/env")
		Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
		Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))
		//Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", workerProcess.Name)))

		usageEvents := lastPageUsageEvents(appName)

		event1 := AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		//event2 := AppUsageEvent{Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(eventsInclude(usageEvents, event1)).To(BeTrue())
		// Expect(eventsInclude(usageEvents, event2)).To(BeTrue())

		StopApp(appGuid)

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", webProcess.Name)))
		//Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", workerProcess.Name)))

		usageEvents = lastPageUsageEvents(appName)
		event1 = AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		//event2 = AppUsageEvent{Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(eventsInclude(usageEvents, event1)).To(BeTrue())
		//	Expect(eventsInclude(usageEvents, event2)).To(BeTrue())

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
	})
})

type Entity struct {
	AppName       string `json:"app_name"`
	AppGuid       string `json:"app_guid"`
	State         string `json:"state"`
	BuildpackName string `json:"buildpack_name"`
	BuildpackGuid string `json:"buildpack_guid"`
	ParentAppName string `json:"parent_app_name"`
	ParentAppGuid string `json:"parent_app_guid"`
	ProcessType   string `json:"process_type"`
}
type AppUsageEvent struct {
	Entity `json:"entity"`
}

type AppUsageEvents struct {
	Resources []AppUsageEvent `struct:"resources"`
}

func eventsInclude(events []AppUsageEvent, event AppUsageEvent) bool {
	found := false
	for _, e := range events {
		found = event.Entity.ParentAppName == e.Entity.ParentAppName &&
			event.Entity.ParentAppGuid == e.Entity.ParentAppGuid &&
			event.Entity.ProcessType == e.Entity.ProcessType &&
			event.Entity.State == e.Entity.State &&
			event.Entity.AppGuid == e.Entity.AppGuid
		if found {
			break
		}
	}
	return found
}

func lastAppUsageEvent(appName string, state string) (bool, AppUsageEvent) {
	var response AppUsageEvents
	cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		cf.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response, DEFAULT_TIMEOUT)
	})

	for _, event := range response.Resources {
		if event.Entity.AppName == appName && event.Entity.State == state {
			return true, event
		}
	}

	return false, AppUsageEvent{}
}

func lastPageUsageEvents(appName string) []AppUsageEvent {
	var response AppUsageEvents

	cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
		cf.ApiRequest("GET", "/v2/app_usage_events?order-direction=desc&page=1", &response, DEFAULT_TIMEOUT)
	})

	return response.Resources
}

type ProcessList struct {
	Processes []Process `json:"resources"`
}

type Process struct {
	Guid    string `json:"guid"`
	Type    string `json:"type"`
	Command string `json:"command"`

	Name string `json:"-"`
}

func getProcess(appGuid, appName string) []Process {
	processesURL := fmt.Sprintf("/v3/apps/%s/processes", appGuid)
	session := cf.Cf("curl", processesURL)
	bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()

	processes := ProcessList{}
	json.Unmarshal(bytes, &processes)

	for i, process := range processes.Processes {
		processes.Processes[i].Name = fmt.Sprintf("v3-%s-%s", appName, process.Type)
	}

	return processes.Processes
}
