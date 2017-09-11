package v3

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

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

			bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
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

			CreateAndMapRoute(destinationAppGuid, TestSetup.RegularUserContext().Space, Config.GetAppsDomain(), webProcess.Name)
			StartApp(destinationAppGuid)
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, webProcess.Name)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))
		})

		It("creates an audit.app.droplet.create event for the copied droplet", func() {
			copyRequestBody := fmt.Sprintf("{\"relationships\": {\"app\": {\"data\": {\"guid\":\"%s\"}}}}", destinationAppGuid)
			copyUrl := fmt.Sprintf("/v3/droplets?source_guid=%s", sourceDropletGuid)
			session := cf.Cf("curl", copyUrl, "-X", "POST", "-d", copyRequestBody)

			bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			var droplet struct {
				Guid string `json:"guid"`
			}
			json.Unmarshal(bytes, &droplet)
			copiedDropletGuid := droplet.Guid

			WaitForDropletToCopy(copiedDropletGuid)

			DeleteApp(appGuid) // to prove that the new app does not depend on the old app

			AssignDropletToApp(destinationAppGuid, copiedDropletGuid)
			eventsQuery := fmt.Sprintf("v2/events?q=type:audit.app.droplet.create&q=actee:%s", destinationAppGuid)
			session = cf.Cf("curl", eventsQuery, "-X", "GET")
			bytes = session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()

			type request struct {
				SourceDropletGuid string `json:"source_droplet_guid"`
			}

			type metadata struct {
				NewDropletGuid string  `json:"droplet_guid"`
				Request        request `json:"request"`
			}

			type entity struct {
				Type      string   `json:"type"`
				Actee     string   `json:"actee"`
				ActeeName string   `json:"actee_name"`
				Metadata  metadata `json:"metadata"`
			}

			type event struct {
				Entity entity `json:"entity"`
			}

			var resources struct {
				Events []event `json:"resources"`
			}

			json.Unmarshal(bytes, &resources)

			Expect(len(resources.Events) > 0).Should(BeTrue())
			Expect(resources.Events).Should(ContainElement(event{
				entity{
					Type:      "audit.app.droplet.create",
					Actee:     destinationAppGuid,
					ActeeName: destinationAppName,
					Metadata: metadata{
						NewDropletGuid: copiedDropletGuid,
						Request: request{
							SourceDropletGuid: sourceDropletGuid,
						},
					},
				},
			}))
		})

	})
})
