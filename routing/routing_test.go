package routing

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Routing", func() {
	var (
		app1              string
		helloRoutingAsset = assets.NewAssets().HelloRouting
	)

	BeforeEach(func() {
		app1 = pushApp(helloRoutingAsset)
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", app1, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("Context paths", func() {
		var (
			app2     string
			app2Path = "/app2"
			app3     string
			app3Path = "/app3/long/sub/path"
			domain   string
		)

		BeforeEach(func() {
			domain = app1
			app2 = pushApp(helloRoutingAsset)
			app3 = pushApp(helloRoutingAsset)

			mapRouteToApp(domain, app2Path, app2)
			mapRouteToApp(domain, app3Path, app3)
		})

		AfterEach(func() {
			deleteApp(app1)
			deleteApp(app2)
			deleteApp(app3)
		})

		Context("when another app has a route with a context path", func() {
			It("routes to app with context path", func() {
				Eventually(func() string {
					return helpers.CurlAppRoot(domain)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring(app1))

				Eventually(func() string {
					return helpers.CurlApp(domain, app2Path)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring(app2))

				Eventually(func() string {
					return helpers.CurlApp(domain, app3Path)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring(app3))
			})
		})
	})
})

func pushApp(asset string) string {
	app := generator.PrefixedRandomName("RATS-APP-")
	Expect(cf.Cf("push", app, "-p", asset).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	return app
}

func deleteApp(appName string) {
	Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func getAppGuid(appName string) string {
	appGuid := cf.Cf("app", appName, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()
	return strings.TrimSpace(string(appGuid))
}

func mapRouteToApp(domain, path, app string) {
	spaceGuid, domainGuid := getSpaceAndDomainGuids(app)

	routeGuid := createRoute(domain, path, spaceGuid, domainGuid)
	appGuid := getAppGuid(app)

	Expect(cf.Cf("curl", "/v2/apps/"+appGuid+"/routes/"+routeGuid, "-X", "PUT").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
}

func createRoute(domainName, contextPath, spaceGuid, domainGuid string) string {
	jsonBody := "{\"host\":\"" + domainName + "\", \"path\":\"" + contextPath + "\", \"domain_guid\":\"" + domainGuid + "\",\"space_guid\":\"" + spaceGuid + "\"}"
	routePostResponseBody := cf.Cf("curl", "/v2/routes", "-X", "POST", "-d", jsonBody).Wait(CF_PUSH_TIMEOUT).Out.Contents()

	var routeResponseJSON struct {
		Metadata struct {
			Guid string `json:"guid"`
		} `json:"metadata"`
	}
	json.Unmarshal([]byte(routePostResponseBody), &routeResponseJSON)
	return routeResponseJSON.Metadata.Guid
}

func getSpaceAndDomainGuids(app string) (string, string) {
	getRoutePath := fmt.Sprintf("/v2/routes?q=host:%s", app)
	routeBody := cf.Cf("curl", getRoutePath).Wait(DEFAULT_TIMEOUT).Out.Contents()
	var routeJSON struct {
		Resources []struct {
			Entity struct {
				SpaceGuid  string `json:"space_guid"`
				DomainGuid string `json:"domain_guid"`
			} `json:"entity"`
		} `json:"resources"`
	}
	json.Unmarshal([]byte(routeBody), &routeJSON)

	spaceGuid := routeJSON.Resources[0].Entity.SpaceGuid
	domainGuid := routeJSON.Resources[0].Entity.DomainGuid

	return spaceGuid, domainGuid
}
