package v3

import (
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

var _ = Describe("v3 buildpack app lifecycle", func() {
	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
		token                           string
		uploadUrl                       string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo"=>"bar"`
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl = fmt.Sprintf("%s%s/v3/packages/%s/upload", config.Protocol(), config.ApiEndpoint, packageGuid)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, config)
		DeleteApp(appGuid)
	})

	Context("with a ruby_buildpack", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		It("can run apps with processes from the Procfile", func() {
			dropletGuid := StageBuildpackPackage(packageGuid, "ruby_buildpack")
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")
			workerProcess := GetProcessByType(processes, "worker")

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

	Context("with a java_buildpack", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().JavaSpringZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		It("can run spring apps", func() {
			dropletGuid := StageBuildpackPackage(packageGuid, "java_buildpack")
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")

			Expect(webProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(appGuid, context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("ok"))

			Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))

			usageEvents := LastPageUsageEvents(context)

			event1 := AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())

			StopApp(appGuid)

			Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+stopped", webProcess.Name)))

			usageEvents = LastPageUsageEvents(context)
			event1 = AppUsageEvent{Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())

			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))
		})
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
			token                           string
		)

		BeforeEach(func() {
			appName = generator.PrefixedRandomName("CATS-APP-")
			spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
			appCreationEnvironmentVariables = `"foo":"bar"`
			appGuid = CreateDockerApp(appName, spaceGuid, `{"foo":"bar"}`)
			packageGuid = CreateDockerPackage(appGuid, "cloudfoundry/diego-docker-app:latest")
			token = GetAuthToken()
		})

		AfterEach(func() {
			FetchRecentLogs(appGuid, token, config)
			DeleteApp(appGuid)
		})

		It("can run apps", func() {
			dropletGuid := StageDockerPackage(packageGuid)
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")

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
