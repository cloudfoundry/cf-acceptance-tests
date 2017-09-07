package v3

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = V3Describe("v3 buildpack app lifecycle", func() {
	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
		token                           string
		uploadUrl                       string
		expectedNullResponse            string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo"=>"bar"`
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl = fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)

		appUrl := "https://" + appName + "." + Config.GetAppsDomain()
		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait(Config.DefaultTimeoutDuration())
		expectedNullResponse = string(nullSession.Buffer().Contents())
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
	})

	Context("with a ruby_buildpack", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		It("can run apps with processes from the Procfile", func() {
			lastUsageEventGuid := LastAppUsageEventGuid(TestSetup)

			buildGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())

			usageEvents := UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
			event := AppUsageEvent{Entity: Entity{State: "STAGING_STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event)).To(BeTrue())

			WaitForBuildToStage(buildGuid)

			usageEvents = UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
			event = AppUsageEvent{Entity: Entity{State: "STAGING_STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event)).To(BeTrue())

			dropletGuid := GetDropletFromBuild(buildGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")
			workerProcess := GetProcessByType(processes, "worker")

			Expect(webProcess.Guid).ToNot(BeEmpty())
			Expect(workerProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(appGuid, TestSetup.RegularUserContext().Space, Config.GetAppsDomain(), webProcess.Name)

			lastUsageEventGuid = LastAppUsageEventGuid(TestSetup)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.CfPushTimeoutDuration()).Should(ContainSubstring("Catnip?"))

			output := helpers.CurlApp(Config, webProcess.Name, "/env")
			Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
			Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", workerProcess.Name)))

			usageEvents = UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)

			event1 := AppUsageEvent{Entity: Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
			event2 := AppUsageEvent{Entity: Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())
			Expect(UsageEventsInclude(usageEvents, event2)).To(BeTrue())

			StopApp(appGuid)

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", webProcess.Name)))
			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", workerProcess.Name)))

			usageEvents = UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
			event1 = AppUsageEvent{Entity: Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
			event2 = AppUsageEvent{Entity: Entity{ProcessType: workerProcess.Type, AppGuid: workerProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())
			Expect(UsageEventsInclude(usageEvents, event2)).To(BeTrue())

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))
		})
	})

	Context("with a java_buildpack", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().JavaSpringZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		It("can run spring apps", func() {
			buildGuid := StageBuildpackPackage(packageGuid, Config.GetJavaBuildpackName())
			WaitForBuildToStage(buildGuid)
			dropletGuid := GetDropletFromBuild(buildGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")

			Expect(webProcess.Guid).ToNot(BeEmpty())
			ScaleProcess(appGuid, webProcess.Type, V3_JAVA_MEMORY_LIMIT)

			CreateAndMapRoute(appGuid, TestSetup.RegularUserContext().Space, Config.GetAppsDomain(), webProcess.Name)

			lastUsageEventGuid := LastAppUsageEventGuid(TestSetup)
			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("ok"))

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

			usageEvents := UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)

			event1 := AppUsageEvent{Entity: Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())

			StopApp(appGuid)

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", webProcess.Name)))

			usageEvents = UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
			event1 = AppUsageEvent{Entity: Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
			Expect(UsageEventsInclude(usageEvents, event1)).To(BeTrue())

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))
		})
	})
})

var _ = V3Describe("v3 docker app lifecycle", func() {
	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
		token                           string
		expectedNullResponse            string
	)

	BeforeEach(func() {
		if !Config.GetIncludeDocker() {
			Skip(skip_messages.SkipDockerMessage)
		}
		appName = random_name.CATSRandomName("APP")
		spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo":"bar"`
		appGuid = CreateDockerApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreateDockerPackage(appGuid, "cloudfoundry/diego-docker-app:latest")
		token = GetAuthToken()

		appUrl := "https://" + appName + "." + Config.GetAppsDomain()
		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait(Config.DefaultTimeoutDuration())
		expectedNullResponse = string(nullSession.Buffer().Contents())
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
	})

	It("can run apps", func() {
		buildGuid := StageDockerPackage(packageGuid)
		WaitForBuildToStage(buildGuid)
		dropletGuid := GetDropletFromBuild(buildGuid)

		AssignDropletToApp(appGuid, dropletGuid)

		processes := GetProcesses(appGuid, appName)
		webProcess := GetProcessByType(processes, "web")

		Expect(webProcess.Guid).ToNot(BeEmpty())

		CreateAndMapRoute(appGuid, TestSetup.RegularUserContext().Space, Config.GetAppsDomain(), webProcess.Name)

		lastUsageEventGuid := LastAppUsageEventGuid(TestSetup)
		StartApp(appGuid)

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, webProcess.Name)
		}, Config.DefaultTimeoutDuration()).Should(Equal("0"))

		output := helpers.CurlApp(Config, webProcess.Name, "/env")
		Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
		Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

		Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
		usageEvents := UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)

		event := AppUsageEvent{Entity: Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STARTED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(UsageEventsInclude(usageEvents, event)).To(BeTrue())

		StopApp(appGuid)

		Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", webProcess.Name)))

		usageEvents = UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
		event = AppUsageEvent{Entity: Entity{ProcessType: webProcess.Type, AppGuid: webProcess.Guid, State: "STOPPED", ParentAppGuid: appGuid, ParentAppName: appName}}
		Expect(UsageEventsInclude(usageEvents, event)).To(BeTrue())

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, webProcess.Name)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))
	})
})
