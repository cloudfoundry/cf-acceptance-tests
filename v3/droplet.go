package v3

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-test-helpers/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = V3Describe("droplet features", func() {

	var (
		appGuid     string
		appName     string
		packageGuid string
		spaceGuid   string
		token       string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		appGuid = CreateApp(appName, spaceGuid, "{}")
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
	})

	Context("copying a droplet", func() {
		var (
			destinationAppGuid string
			destinationAppName string
			sourceDropletGuid  string
		)

		SkipOnK8s("App droplets not supported")

		BeforeEach(func() {
			buildGuid := StageBuildpackPackage(packageGuid, Config.GetRubyBuildpackName())
			WaitForBuildToStage(buildGuid)
			sourceDropletGuid = GetDropletFromBuild(buildGuid)

			destinationAppName = random_name.CATSRandomName("APP")
			destinationAppGuid = CreateApp(destinationAppName, spaceGuid, "{}")
		})

		It("can copy a droplet", func() {
			copyRequestBody := fmt.Sprintf("{\"relationships\": {\"app\": {\"data\": {\"guid\":\"%s\"}}}}", destinationAppGuid)
			copyUrl := fmt.Sprintf("/v3/droplets?source_guid=%s", sourceDropletGuid)
			session := cf.Cf("curl", copyUrl, "-X", "POST", "-d", copyRequestBody)

			bytes := session.Wait().Out.Contents()
			var droplet struct {
				Guid string `json:"guid"`
			}
			json.Unmarshal(bytes, &droplet)
			copiedDropletGuid := droplet.Guid

			WaitForDropletToCopy(copiedDropletGuid)

			DeleteApp(appGuid) // to prove that the new app does not depend on the old app

			AssignDropletToApp(destinationAppGuid, copiedDropletGuid)

			processes := GetProcesses(destinationAppGuid, destinationAppName)
			webProcess := GetProcessByType(processes, "web")
			workerProcess := GetProcessByType(processes, "worker")

			Expect(webProcess.Guid).ToNot(BeEmpty())
			Expect(workerProcess.Guid).ToNot(BeEmpty())

			CreateAndMapRoute(destinationAppGuid, Config.GetAppsDomain(), webProcess.Name)
			StartApp(destinationAppGuid)
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}).Should(ContainSubstring("Hi, I'm Dora!"))
		})

		It("creates an audit.app.droplet.create event for the copied droplet", func() {
			copyRequestBody := fmt.Sprintf("{\"relationships\": {\"app\": {\"data\": {\"guid\":\"%s\"}}}}", destinationAppGuid)
			copyUrl := fmt.Sprintf("/v3/droplets?source_guid=%s", sourceDropletGuid)
			session := cf.Cf("curl", copyUrl, "-X", "POST", "-d", copyRequestBody)

			bytes := session.Wait().Out.Contents()
			var droplet struct {
				Guid string `json:"guid"`
			}
			json.Unmarshal(bytes, &droplet)
			copiedDropletGuid := droplet.Guid

			WaitForDropletToCopy(copiedDropletGuid)

			DeleteApp(appGuid) // to prove that the new app does not depend on the old app

			AssignDropletToApp(destinationAppGuid, copiedDropletGuid)
			eventsQuery := fmt.Sprintf("v3/audit_events?types=audit.app.droplet.create&target_guids=%s", destinationAppGuid)
			session = cf.Cf("curl", eventsQuery, "-X", "GET")
			bytes = session.Wait().Out.Contents()

			type target struct {
				Type string `json:"type"`
				Guid string `json:"guid"`
				Name string `json:"name"`
			}

			type event struct {
				Guid   string `json:"guid"`
				Type   string `json:"type"`
				Target target `json:"target"`
			}

			var resources struct {
				Events []event `json:"resources"`
			}

			json.Unmarshal(bytes, &resources)

			Expect(len(resources.Events) == 1).Should(BeTrue())
			Expect(resources.Events[0].Target.Type).Should(Equal("app"))
			Expect(resources.Events[0].Target.Guid).Should(Equal(destinationAppGuid))
			Expect(resources.Events[0].Target.Name).Should(Equal(destinationAppName))
		})

	})
})
