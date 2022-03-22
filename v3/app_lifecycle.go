package v3

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
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
		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
		expectedNullResponse = string(nullSession.Buffer().Contents())
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		DeleteApp(appGuid)
	})

	Context("with a ruby_buildpack", func() {
		BeforeEach(func() {
			UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
			WaitForPackageToBeReady(packageGuid)
		})

		It("can run apps with processes from the Procfile", func() {
			lastUsageEventGuid := app_helpers.LastAppUsageEventGuid(TestSetup)

			buildGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())

			usageEvents := app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)
			event := app_helpers.AppUsageEvent{}
			event.State.Current = "STAGING_STARTED"
			event.App.Guid = appGuid
			event.App.Name = appName
			Expect(app_helpers.UsageEventsInclude(usageEvents, event)).To(BeTrue())

			WaitForBuildToStage(buildGuid)

			usageEvents = app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)
			event = app_helpers.AppUsageEvent{}
			event.State.Current = "STAGING_STOPPED"
			event.App.Guid = appGuid
			event.App.Name = appName
			Expect(app_helpers.UsageEventsInclude(usageEvents, event)).To(BeTrue())

			dropletGuid := GetDropletFromBuild(buildGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess := GetProcessByType(processes, "web")
			workerProcess := GetProcessByType(processes, "worker")

			Expect(webProcess.Guid).ToNot(BeEmpty())
			Expect(workerProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(appGuid, Config.GetAppsDomain(), webProcess.Name)

			lastUsageEventGuid = app_helpers.LastAppUsageEventGuid(TestSetup)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.CfPushTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

			output := helpers.CurlApp(Config, webProcess.Name, "/env")
			Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
			Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", workerProcess.Name)))

			usageEvents = app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)

			event1 := app_helpers.AppUsageEvent{}
			event1.Process.Type = webProcess.Type
			event1.Process.Guid = webProcess.Guid
			event1.State.Current = "STARTED"
			event1.App.Guid = appGuid
			event1.App.Name = appName

			event2 := app_helpers.AppUsageEvent{}
			event2.Process.Type = workerProcess.Type
			event2.Process.Guid = workerProcess.Guid
			event2.State.Current = "STARTED"
			event2.App.Guid = appGuid
			event2.App.Name = appName

			Expect(app_helpers.UsageEventsInclude(usageEvents, event1)).To(BeTrue())
			Expect(app_helpers.UsageEventsInclude(usageEvents, event2)).To(BeTrue())

			StopApp(appGuid)

			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", webProcess.Name)))
			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", workerProcess.Name)))

			usageEvents = app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)
			event1 = app_helpers.AppUsageEvent{}
			event1.Process.Type = webProcess.Type
			event1.Process.Guid = webProcess.Guid
			event1.State.Current = "STOPPED"
			event1.App.Guid = appGuid
			event1.App.Name = appName

			event2 = app_helpers.AppUsageEvent{}
			event2.Process.Type = workerProcess.Type
			event2.Process.Guid = workerProcess.Guid
			event2.State.Current = "STOPPED"
			event2.App.Guid = appGuid
			event2.App.Name = appName

			Expect(app_helpers.UsageEventsInclude(usageEvents, event1)).To(BeTrue())
			Expect(app_helpers.UsageEventsInclude(usageEvents, event2)).To(BeTrue())

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}).Should(ContainSubstring(expectedNullResponse))
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

			CreateAndMapRoute(appGuid, Config.GetAppsDomain(), webProcess.Name)

			lastUsageEventGuid := app_helpers.LastAppUsageEventGuid(TestSetup)
			StartApp(appGuid)

			// Because v3 start returns immediately, the curl returning "ok" is the signal that Push has finished
			// So we're using the Push timeout here
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.CfPushTimeoutDuration()).Should(ContainSubstring("ok"))

			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

			usageEvents := app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)

			event1 := app_helpers.AppUsageEvent{}
			event1.Process.Type = webProcess.Type
			event1.Process.Guid = webProcess.Guid
			event1.State.Current = "STARTED"
			event1.App.Guid = appGuid
			event1.App.Name = appName
			Expect(app_helpers.UsageEventsInclude(usageEvents, event1)).To(BeTrue())

			StopApp(appGuid)

			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", webProcess.Name)))

			usageEvents = app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)
			event1 = app_helpers.AppUsageEvent{}
			event1.Process.Type = webProcess.Type
			event1.Process.Guid = webProcess.Guid
			event1.State.Current = "STOPPED"
			event1.App.Guid = appGuid
			event1.App.Name = appName
			Expect(app_helpers.UsageEventsInclude(usageEvents, event1)).To(BeTrue())

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}).Should(ContainSubstring(expectedNullResponse))
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
		packageGuid = CreateDockerPackage(appGuid, Config.GetPublicDockerAppImage())

		appUrl := "https://" + appName + "." + Config.GetAppsDomain()
		nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
		expectedNullResponse = string(nullSession.Buffer().Contents())
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
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

		CreateAndMapRoute(appGuid, Config.GetAppsDomain(), webProcess.Name)

		lastUsageEventGuid := app_helpers.LastAppUsageEventGuid(TestSetup)
		StartApp(appGuid)

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, webProcess.Name)
		}).Should(Equal("0"))

		output := helpers.CurlApp(Config, webProcess.Name, "/env")
		Expect(output).To(ContainSubstring(fmt.Sprintf("application_name\\\":\\\"%s", appName)))
		Expect(output).To(ContainSubstring(appCreationEnvironmentVariables))

		Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
		usageEvents := app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)

		event := app_helpers.AppUsageEvent{}
		event.Process.Type = webProcess.Type
		event.Process.Guid = webProcess.Guid
		event.State.Current = "STARTED"
		event.App.Guid = appGuid
		event.App.Name = appName
		Expect(app_helpers.UsageEventsInclude(usageEvents, event)).To(BeTrue())

		StopApp(appGuid)

		Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(stopped)", webProcess.Name)))

		usageEvents = app_helpers.UsageEventsAfterGuid(lastUsageEventGuid)
		event = app_helpers.AppUsageEvent{}
		event.Process.Type = webProcess.Type
		event.Process.Guid = webProcess.Guid
		event.State.Current = "STOPPED"
		event.App.Guid = appGuid
		event.App.Name = appName
		Expect(app_helpers.UsageEventsInclude(usageEvents, event)).To(BeTrue())

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, webProcess.Name)
		}).Should(ContainSubstring(expectedNullResponse))
	})
})
