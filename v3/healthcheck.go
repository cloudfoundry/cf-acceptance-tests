package v3

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = V3Describe("Healthcheck", func() {
	var (
		appName    string
		appGuid    string
		token      string
		webProcess Process
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName := TestSetup.RegularUserContext().Space
		spaceGuid := GetSpaceGuidFromName(spaceName)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid := CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)

		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)

		buildGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGuid)
		dropletGuid := GetDropletFromBuild(buildGuid)
		AssignDropletToApp(appGuid, dropletGuid)
		processes := GetProcesses(appGuid, appName)
		webProcess = GetProcessByType(processes, "web")
		CreateAndMapRoute(appGuid, spaceName, Config.GetAppsDomain(), appName)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
	})

	Context("when the healthcheck is set to process", func() {
		It("starts up successfully", func() {
			updateProcessPath := fmt.Sprintf("/v3/processes/%s", webProcess.Guid)
			setHealthCheckBody := `{"health_check": {"type": "process"}}`

			Expect(cf.Cf("curl", updateProcessPath, "-X", "PATCH", "-d", setHealthCheckBody).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			StartApp(appGuid)

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration(), 2*time.Second).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})

	Context("when the healthcheck is set to port", func() {
		It("starts up successfully", func() {
			updateProcessPath := fmt.Sprintf("/v3/processes/%s", webProcess.Guid)
			setHealthCheckBody := `{"health_check": {"type": "port"}}`

			Expect(cf.Cf("curl", updateProcessPath, "-X", "PATCH", "-d", setHealthCheckBody).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			StartApp(appGuid)

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration(), 2*time.Second).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})

	Context("when the healthcheck is set to http", func() {
		It("starts up successfully", func() {
			updateProcessPath := fmt.Sprintf("/v3/processes/%s", webProcess.Guid)
			setHealthCheckBody := `{"health_check": {"type": "http", "data":{"endpoint":"/health"}}}`

			Expect(cf.Cf("curl", updateProcessPath, "-X", "PATCH", "-d", setHealthCheckBody).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			StartApp(appGuid)

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration(), 2*time.Second).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})
})
