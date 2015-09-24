package routing

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"testing"
)

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

func PushApp(asset string) string {
	app := generator.PrefixedRandomName("RATS-APP-")
	Expect(cf.Cf("push", app, "-m", "128M", "-p", asset, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	return app
}

func ScaleAppInstances(appName string, instances int) {
	Expect(cf.Cf("scale", appName, "-i", strconv.Itoa(instances)).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	Eventually(func() string {
		return string(cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT).Out.Contents())
	}, DEFAULT_TIMEOUT*2, 2*time.Second).
		Should(ContainSubstring(fmt.Sprintf("instances: %d/%d", instances, instances)))
}

func DeleteApp(appName string) {
	Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func GetAppGuid(appName string) string {
	appGuid := cf.Cf("app", appName, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()
	return strings.TrimSpace(string(appGuid))
}

func MapRouteToApp(domain, path, app string) {
	spaceGuid, domainGuid := GetSpaceAndDomainGuids(app)

	routeGuid := CreateRoute(domain, path, spaceGuid, domainGuid)
	appGuid := GetAppGuid(app)

	Expect(cf.Cf("curl", "/v2/apps/"+appGuid+"/routes/"+routeGuid, "-X", "PUT").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
}

func CreateRoute(domainName, contextPath, spaceGuid, domainGuid string) string {
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

func GetSpaceAndDomainGuids(app string) (string, string) {
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

func GetAppInfo(appName string) (host, port string) {
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

func RegisterRoute(appRoute string, ip string, port string, routeServiceName string) {
	systemDomain := config.SystemDomain
	oauthPassword := config.ClientSecret
	oauthUrl := config.Protocol() + "uaa." + systemDomain
	routingApiEndpoint := config.Protocol() + "api." + systemDomain

	appsDomain := config.AppsDomain
	routeServiceRoute := "https://" + routeServiceName + "." + appsDomain
	route := appRoute + "." + appsDomain
	routeJSON := `[{"route":"` + route + `","port":` + port + `,"ip":"` + ip + `","ttl":60, "route_service_url":"` + routeServiceRoute + `"}]`

	args := []string{"register", routeJSON, "--api", routingApiEndpoint, "--client-id", "gorouter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
	session := Rtr(args...)
	Eventually(session.Out).Should(Say("Successfully registered routes"))
}

func Rtr(args ...string) *Session {
	session, err := Start(exec.Command("rtr", args...), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}

const (
	DEFAULT_TIMEOUT = 30 * time.Second
	CF_PUSH_TIMEOUT = 2 * time.Minute
)

var config helpers.Config

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	config = helpers.LoadConfig()

	componentName := "Routing"

	rs := []Reporter{}

	context := helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
		Expect(config.SystemDomain).ToNot(Equal(""), "Must provide a system domain for the routing suite")
		Expect(config.ClientSecret).ToNot(Equal(""), "Must provide a client secret for the routing suite")
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}
