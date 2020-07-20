package routing

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"path/filepath"

	"encoding/json"
	"regexp"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = RoutingDescribe("Multiple App Ports", func() {
	var (
		app               string
		secondRoute       string
		multiPortAppAsset = assets.NewAssets().MultiPortApp
	)

	BeforeEach(func() {
		app = random_name.CATSRandomName("APP")
		cmd := fmt.Sprintf("go-online --ports=7777,8888,8080")

		Expect(cf.Cf("push",
			app,
			"-b", Config.GetGoBuildpackName(),
			"-c", cmd,
			"-f", filepath.Join(multiPortAppAsset, "manifest.yml"),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", multiPortAppAsset,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(app)
		Expect(cf.Cf("delete", app, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("when app only has single route", func() {
		Context("when no ports are specified for the app", func() {
			It("should listen on the default app port", func() {
				Eventually(func() string {
					return helpers.CurlApp(Config, app, "/port")
				}).Should(ContainSubstring("8080"))
			})
		})
	})

	Context("when app has multiple ports mapped", func() {
		BeforeEach(func() {
			appGuid := app_helpers.GetAppGuid(app)
			Expect(cf.Cf(
				"curl",
				fmt.Sprintf("/v2/apps/%s", appGuid),
				"-X", "PUT", "-d", `{"ports": [7777,8888,8080]}`,
			).Wait()).To(Exit(0))

			// create 2nd route
			secondRoute = fmt.Sprintf("%s-two", app)
			Expect(cf.Cf("create-route", Config.GetAppsDomain(),
				"--hostname", secondRoute,
			).Wait()).To(Exit(0))
			// map app route to other port
			createRouteMapping(app, secondRoute, 7777)
		})

		It("should listen on multiple ports", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, secondRoute, "/port")
			}).Should(ContainSubstring("7777"))

			Consistently(func() string {
				return helpers.CurlApp(Config, app, "/port")
			}, Config.SleepTimeoutDuration(), "5s").Should(ContainSubstring("8080"))
		})
	})
})

func getRouteGuid(hostname string) string {
	routeQuery := fmt.Sprintf("/v2/routes?q=host:%s", hostname)
	getRoutesCurl := cf.Cf("curl", routeQuery)
	Expect(getRoutesCurl.Wait()).To(Exit(0))

	routeGuidRegex := regexp.MustCompile(`\s+"guid": "(.+)"`)
	return routeGuidRegex.FindStringSubmatch(string(getRoutesCurl.Out.Contents()))[1]
}

func createRouteMapping(appName string, hostname string, appPort uint16) {
	appGuid := app_helpers.GetAppGuid(appName)
	routeGuid := getRouteGuid(hostname)

	bodyMap := map[string]interface{}{
		"app_guid":   appGuid,
		"route_guid": routeGuid,
		"app_port":   appPort,
	}

	data, err := json.Marshal(bodyMap)
	Expect(err).ToNot(HaveOccurred())

	Expect(cf.Cf("curl", fmt.Sprintf("/v2/route_mappings"), "-X", "POST", "-d", string(data)).Wait()).To(Exit(0))
}
