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

type ProcessStats struct {
	Instance struct {
		State string `json:"state"`
	} `json:"0"`
}

var _ = Describe("process", func() {
	var (
		appName     string
		appGuid     string
		packageGuid string
		spaceGuid   string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token := GetAuthToken()
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
	})

	Describe("terminating an instance", func() {
		var (
			index       = 0
			processType = "web"
			webProcess  Process
		)

		BeforeEach(func() {
			dropletGuid := StagePackage(packageGuid, "{}")
			WaitForDropletToStage(dropletGuid)

			AssignDropletToApp(appGuid, dropletGuid)

			processes := getProcess(appGuid, appName)
			for _, process := range processes {
				if process.Type == "web" {
					webProcess = process
				}
			}

			CreateAndMapRoute(appGuid, context.RegularUserContext().Space, helpers.LoadConfig().AppsDomain, webProcess.Name)

			StartApp(appGuid)

			Eventually(func() string {
				return helpers.CurlAppRoot(webProcess.Name)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

			Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))
		})

		It("/v3/apps/:guid/processes/:guid/instances/:index", func() {
			statsUrl := fmt.Sprintf("/v2/apps/%s/stats", webProcess.Guid)
			statsBody := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
			statsJSON := ProcessStats{}
			json.Unmarshal(statsBody, &statsJSON)

			Expect(statsJSON.Instance.State).To(Equal("RUNNING"))

			terminateUrl := fmt.Sprintf("/v3/apps/%s/processes/%s/instances/%d", appGuid, processType, index)
			cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait(DEFAULT_TIMEOUT)

			Eventually(func() string {
				statsBodyAfter := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
				json.Unmarshal(statsBodyAfter, &statsJSON)
				return statsJSON.Instance.State
			}).Should(Equal("DOWN"))
		})

		It("/v3/processes/:guid/instances/:index", func() {
			statsUrl := fmt.Sprintf("/v2/apps/%s/stats", webProcess.Guid)
			statsBody := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
			statsJSON := ProcessStats{}
			json.Unmarshal(statsBody, &statsJSON)

			Expect(statsJSON.Instance.State).To(Equal("RUNNING"))

			terminateUrl := fmt.Sprintf("/v3/processes/%s/instances/%d", webProcess.Guid, index)
			cf.Cf("curl", terminateUrl, "-X", "DELETE").Wait(DEFAULT_TIMEOUT)

			Eventually(func() string {
				statsBodyAfter := cf.Cf("curl", statsUrl).Wait(DEFAULT_TIMEOUT).Out.Contents()
				json.Unmarshal(statsBodyAfter, &statsJSON)
				return statsJSON.Instance.State
			}).Should(Equal("DOWN"))
		})
	})
})
