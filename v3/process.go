package v3

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type ProcessStats struct {
	Instance []struct {
		State string `json:"state"`
	} `json:"resources"`
}

var _ = V3Describe("process", func() {
	var (
		appName     string
		appGuid     string
		packageGuid string
		spaceGuid   string
		token       string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
	})

	Describe("terminating an instance", func() {
		var (
			index       = 0
			processType = "web"
			webProcess  Process
		)

		BeforeEach(func() {
			dropletGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess = GetProcessByType(processes, "web")

			CreateAndMapRoute(appGuid, TestSetup.RegularUserContext().Space, Config.GetAppsDomain(), webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

			Expect(string(cf.Cf("apps").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
		})

		Context("/v3/apps/:guid/processes/:type/instances/:index", func() {
			It("restarts the instance", func() {
				statsUrl := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

				By("ensuring the instance is running")
				statsBody := cf.Cf("curl", statsUrl).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				statsJSON := ProcessStats{}
				json.Unmarshal(statsBody, &statsJSON)
				Expect(statsJSON.Instance[0].State).To(Equal("RUNNING"))

				By("terminating the instance")
				terminateUrl := fmt.Sprintf("/v3/apps/%s/processes/%s/instances/%d", appGuid, processType, index)
				cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait(Config.DefaultTimeoutDuration())

				By("ensuring the instance is no longer running")
				// Note that this depends on a 30s run loop waking up in Diego.
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 45*time.Second).ShouldNot(Equal("RUNNING"))

				By("ensuring the instance is running again")
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 45*time.Second).Should(Equal("RUNNING"))
			})
		})

		Context("/v3/processes/:guid/instances/:index", func() {
			It("restarts the instance", func() {
				statsUrl := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

				By("ensuring the instance is running")
				statsBody := cf.Cf("curl", statsUrl).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				statsJSON := ProcessStats{}
				json.Unmarshal(statsBody, &statsJSON)
				Expect(statsJSON.Instance[0].State).To(Equal("RUNNING"))

				By("terminating the instance")
				terminateUrl := fmt.Sprintf("/v3/processes/%s/instances/%d", webProcess.Guid, index)
				cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait(Config.DefaultTimeoutDuration())

				By("ensuring the instance is no longer running")
				// Note that this depends on a 30s run loop waking up in Diego.
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 45*time.Second).ShouldNot(Equal("RUNNING"))

				By("ensuring the instance is running again")
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 45*time.Second).Should(Equal("RUNNING"))
			})
		})
	})
})
