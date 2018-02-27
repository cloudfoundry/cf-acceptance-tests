package capi_experimental

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = CapiExperimentalDescribe("apply_manifest", func() {
	var (
		appName     string
		appGuid     string
		packageGuid string
		spaceGuid   string
		spaceName   string
		token       string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGuid = GetSpaceGuidFromName(spaceName)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)

		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)

		buildGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGuid)
		dropletGuid := GetDropletFromBuild(buildGuid)
		AssignDropletToApp(appGuid, dropletGuid)

		CreateRoute(spaceName, Config.GetAppsDomain(), appName)
		StartApp(appGuid)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
	})

	Describe("Applying manifest to existing app", func() {
		var (
			manifest string
			endpoint string
		)

		BeforeEach(func() {
			endpoint = fmt.Sprintf("/v3/apps/%s/actions/apply_manifest", appGuid)
		})

		Context("when scaling the web process", func() {
			BeforeEach(func() {
				manifest = `
applications:
- instances: 2
  memory: 300M
`
			})

			It("successfully completes the job", func() {
				session := cf.Cf("curl", endpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifest, "-i")
				Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				response := session.Out.Contents()
				Expect(string(response)).To(ContainSubstring("202 Accepted"))

				PollJob(GetJobPath(response))
			})
		})
	})
})
