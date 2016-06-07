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
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("route_mapping", func() {
	type RouteList struct {
		Resources []struct {
			Metadata struct {
				Guid string `json:"guid"`
			} `json:"metadata"`
		} `json:"resources"`
	}

	var (
		appName     string
		appGuid     string
		packageGuid string
		spaceGuid   string
		spaceName   string
		token       string
		webProcess  Process
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceName = context.RegularUserContext().Space
		spaceGuid = GetSpaceGuidFromName(spaceName)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", config.Protocol(), config.ApiEndpoint, packageGuid)

		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)

		dropletGuid := StageBuildpackPackage(packageGuid, "ruby_buildpack")
		WaitForDropletToStage(dropletGuid)
		AssignDropletToApp(appGuid, dropletGuid)

		processes := GetProcesses(appGuid, appName)
		webProcess = GetProcessByType(processes, "web")

		CreateRoute(spaceName, helpers.LoadConfig().AppsDomain, appName)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, config)
		DeleteApp(appGuid)
	})

	Describe("Route mapping lifecycle", func() {
		It("creates a route mapping on a specified port", func() {
			updateProcessPath := fmt.Sprintf("/v3/processes/%s", webProcess.Guid)
			setPortBody := `{"ports": [1234], "health_check": {"type": "process"}}`

			Expect(cf.Cf("curl", updateProcessPath, "-X", "PATCH", "-d", setPortBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			getRoutePath := fmt.Sprintf("/v2/routes?q=host:%s", appName)
			routeBody := cf.Cf("curl", getRoutePath).Wait(DEFAULT_TIMEOUT).Out.Contents()

			var routeJSON RouteList
			json.Unmarshal([]byte(routeBody), &routeJSON)
			routeGuid := routeJSON.Resources[0].Metadata.Guid
			addRouteBody := fmt.Sprintf(`
				{
					"relationships": {
						"app":   {"guid": "%s"},
						"route": {"guid": "%s"}
					},
					"app_port": 1234
				}`, appGuid, routeGuid)

			Expect(cf.Cf("curl", "/v3/route_mappings", "-X", "POST", "-d", addRouteBody).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			StartApp(appGuid)
			Expect(cf.Cf("apps").Wait(DEFAULT_TIMEOUT)).To(Say(fmt.Sprintf("%s\\s+started", webProcess.Name)))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})
})
