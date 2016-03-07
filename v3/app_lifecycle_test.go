package v3

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gbytes"
)

var _ = Describe("v3 buildpack app lifecycle", func() {
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

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
		DeleteApp(appGuid)
	})

	It("can run apps with processes from the Procfile", func() {
		dropletGuid := StagePackage(packageGuid, "{}")
		WaitForDropletToStage(dropletGuid)

		AssignDropletToApp(appGuid, dropletGuid)

		var webProcess Process
		var workerProcess Process
		processes := getProcess(appGuid, appName)
		for _, process := range processes {
			if process.Type == "web" {
				webProcess = process
			} else if process.Type == "worker" {
				workerProcess = process
			}
		}

		Expect(webProcess.Guid).ToNot(BeEmpty())
		Expect(workerProcess.Guid).ToNot(BeEmpty())

		CreateAndMapRoute(appGuid, context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, webProcess.Name)

		StartApp(appGuid)

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

		output := helpers.CurlApp(webProcess.Name, "/env")
		Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
		Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))
		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", workerProcess.Name)))

		usageEvents := LastPageUsageEvents(context)

		event1 := AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		event2 := AppUsageEvent{Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())
		Expect(UsageEventsInclude(usageEvents, event2)).To(BeTrue())

		StopApp(appGuid)

		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", webProcess.Name)))
		Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", workerProcess.Name)))

		usageEvents = LastPageUsageEvents(context)
		event1 = AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		event2 = AppUsageEvent{Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())
		Expect(UsageEventsInclude(usageEvents, event2)).To(BeTrue())

		Eventually(func() string {
			return helpers.CurlAppRoot(webProcess.Name)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
	})
})

var _ = Describe("v3 docker app lifecycle", func() {
	config := helpers.LoadConfig()
	if config.IncludeDiegoDocker {
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
			appCreationEnvironmentVariables = `"foo":"bar"`
			appGuid = CreateDockerApp(appName, spaceGuid, `{"foo":"bar"}`)
			packageGuid = CreateDockerPackage(appGuid, "cloudfoundry/diego-docker-app:latest")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			DeleteApp(appGuid)
		})

		It("can run apps", func() {
			dropletGuid := StagePackage(packageGuid, "{}")
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			var webProcess Process
			processes := getProcess(appGuid, appName)
			for _, process := range processes {
				if process.Type == "web" {
					webProcess = process
				}
			}

			Expect(webProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(appGuid, context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(Equal("0"))

			output := helpers.CurlApp(webProcess.Name, "/env")
			Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
			Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

			Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))

			usageEvents := LastPageUsageEvents(context)

			event := AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event)).To(BeTrue())

			StopApp(appGuid)

			Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", webProcess.Name)))

			usageEvents = LastPageUsageEvents(context)
			event = AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event)).To(BeTrue())

			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
		})
	}
})

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
