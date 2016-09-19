package v3

import (
	"encoding/json"
	"fmt"
	"time"

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
		spaceGuid = GetSpaceGuidFromName(testSetup.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", config.Protocol(), config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, config)
		DeleteApp(appGuid)
	})

	Describe("terminating an instance", func() {
		var (
			index       = 0
			processType = "web"
			webProcess  Process
		)

		BeforeEach(func() {
			dropletGuid := StageBuildpackPackage(packageGuid, config.RubyBuildpackName)
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := GetProcesses(appGuid, appName)
			webProcess = GetProcessByType(processes, "web")

			CreateAndMapRoute(appGuid, testSetup.RegularUserContext().Space, config.AppsDomain, webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

			Expect(string(cf.Cf("apps").Wait(DEFAULT_TIMEOUT).Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))
		})

		Context("/v3/apps/:guid/processes/:type/instances/:index", func() {
			It("restarts the instance", func() {
				statsUrl := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

				By("ensuring the instance is running")
				statsBody := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
				statsJSON := ProcessStats{}
				json.Unmarshal(statsBody, &statsJSON)
				Expect(statsJSON.Instance[0].State).To(Equal("RUNNING"))

				By("terminating the instance")
				terminateUrl := fmt.Sprintf("/v3/apps/%s/processes/%s/instances/%d", appGuid, processType, index)
				cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait(DEFAULT_TIMEOUT)

				By("ensuring the instance is no longer running")
				// Note that this depends on a 30s run loop waking up in Diego.
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 35*time.Second).ShouldNot(Equal("RUNNING"))

				By("ensuring the instance is running again")
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 35*time.Second).Should(Equal("RUNNING"))
			})
		})

		Context("/v3/processes/:guid/instances/:index", func() {
			It("restarts the instance", func() {
				statsUrl := fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGuid)

				By("ensuring the instance is running")
				statsBody := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
				statsJSON := ProcessStats{}
				json.Unmarshal(statsBody, &statsJSON)
				Expect(statsJSON.Instance[0].State).To(Equal("RUNNING"))

				By("terminating the instance")
				terminateUrl := fmt.Sprintf("/v3/processes/%s/instances/%d", webProcess.Guid, index)
				cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait(DEFAULT_TIMEOUT)

				By("ensuring the instance is no longer running")
				// Note that this depends on a 30s run loop waking up in Diego.
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 35*time.Second).ShouldNot(Equal("RUNNING"))

				By("ensuring the instance is running again")
				Eventually(func() string {
					statsBodyAfter := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
					json.Unmarshal(statsBodyAfter, &statsJSON)
					return statsJSON.Instance[0].State
				}, 35*time.Second).Should(Equal("RUNNING"))
			})
		})
	})
})
