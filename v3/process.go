package v3

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
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
		app_helpers.AppReport(appName)
		DeleteApp(appGuid)
	})

	Describe("terminating an instance", func() {
		var (
			index       = 0
			processType = "web"
			webProcess  Process
		)

		BeforeEach(func() {
			buildGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())
			WaitForBuildToStage(buildGuid)
			dropletGuid := GetDropletFromBuild(buildGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess = GetProcessByType(processes, "web")

			CreateAndMapRoute(appGuid, Config.GetAppsDomain(), webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}).Should(ContainSubstring("Hi, I'm Dora!"))

			Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
		})

		Context("/v3/apps/:guid/processes/:type/instances/:index", func() {
			It("restarts the instance", func() {
				statsUrl := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

				By("ensuring the instance is running")
				statsBody := cf.Cf("curl", statsUrl).Wait().Out.Contents()
				statsJSON := ProcessStats{}
				json.Unmarshal(statsBody, &statsJSON)
				Expect(statsJSON.Instance[0].State).To(Equal("RUNNING"))

				By("terminating the instance")
				terminateUrl := fmt.Sprintf("/v3/apps/%s/processes/%s/instances/%d", appGuid, processType, index)
				cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait()

				By("ensuring the instance is no longer running")
				// Note that this depends on a 30s run loop waking up in Diego.
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait().Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, V3_PROCESS_TIMEOUT, 1*time.Second).ShouldNot(Equal("RUNNING"))

				By("ensuring the instance is running again")
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait().Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, V3_PROCESS_TIMEOUT, 1*time.Second).Should(Equal("RUNNING"))
			})
		})

		Context("/v3/processes/:guid/instances/:index", func() {
			It("restarts the instance", func() {
				statsUrl := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

				By("ensuring the instance is running")
				statsBody := cf.Cf("curl", statsUrl).Wait().Out.Contents()
				statsJSON := ProcessStats{}
				json.Unmarshal(statsBody, &statsJSON)
				Expect(statsJSON.Instance[0].State).To(Equal("RUNNING"))

				By("terminating the instance")
				terminateUrl := fmt.Sprintf("/v3/processes/%s/instances/%d", webProcess.Guid, index)
				cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait()

				By("ensuring the instance is no longer running")
				// Note that this depends on a 30s run loop waking up in Diego.
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait().Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, V3_PROCESS_TIMEOUT, 1*time.Second).ShouldNot(Equal("RUNNING"))

				By("ensuring the instance is running again")
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait().Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, V3_PROCESS_TIMEOUT, 1*time.Second).Should(Equal("RUNNING"))
			})
		})
	})
})
