package routing

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Route Services", func() {

	Context("when an app has a route service bound", func() {
		var (
			appName          string
			appRoute         string
			routeServiceName string
		)

		BeforeEach(func() {
			// push app
			appName = generator.RandomName()
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Golang).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			// push routing service
			routeServiceName = generator.RandomName()
			Expect(cf.Cf("push", routeServiceName, "-p", assets.NewAssets().LoggingRouteServiceZip).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			// get app info
			appIp, appPort := getAppInfo(appName)

			// associate routing service with app
			systemDomain := config.SystemDomain
			oauthPassword := config.ClientSecret
			oauthUrl := "http://uaa." + systemDomain
			routingApiEndpoint := "http://routing-api." + systemDomain

			appsDomain := config.AppsDomain
			routeServiceRoute := "https://" + routeServiceName + "." + appsDomain
			appRoute = generator.RandomName()
			route := appRoute + "." + appsDomain
			routeJSON := `[{"route":"` + route + `","port":` + appPort + `,"ip":"` + appIp + `","ttl":60, "route_service_url":"` + routeServiceRoute + `"}]`

			args := []string{"register", routeJSON, "--api", routingApiEndpoint, "--client-id", "gorouter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session := Rtr(args...)
			Eventually(session.Out).Should(Say("Successfully registered routes"))
		})

		It("a request to the app is routed through the route service", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(appRoute)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))

			Eventually(func() *Session {
				logs := cf.Cf("logs", "--recent", routeServiceName)
				Expect(logs.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				return logs
			}, DEFAULT_TIMEOUT).Should(Say("Response Body: go, world"))
		})
	})
})

type AppResource struct {
	Metadata struct {
		Url string
	}
}
type AppsResponse struct {
	Resources []AppResource
}
type Stat struct {
	Stats struct {
		Host string
		Port int
	}
}
type StatsResponse map[string]Stat

func getAppInfo(appName string) (host, port string) {
	var appsResponse AppsResponse
	var statsResponse StatsResponse

	cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
	json.Unmarshal(cfResponse, &appsResponse)
	serverAppUrl := appsResponse.Resources[0].Metadata.Url

	cfResponse = cf.Cf("curl", fmt.Sprintf("%s/stats", serverAppUrl)).Wait(DEFAULT_TIMEOUT).Out.Contents()
	json.Unmarshal(cfResponse, &statsResponse)

	appIp := statsResponse["0"].Stats.Host
	appPort := fmt.Sprintf("%d", statsResponse["0"].Stats.Port)
	return appIp, appPort
}
