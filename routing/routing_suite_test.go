package routing

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"

	"testing"
)

const (
	DEFAULT_MEMORY_LIMIT = "256M"
	deaUnsupportedTag    = "{NO_DEA_SUPPORT} "
)

var (
	DEFAULT_TIMEOUT = 30 * time.Second
	CF_PUSH_TIMEOUT = 2 * time.Minute

	context helpers.SuiteContext
	config  helpers.Config
)

type Metadata struct {
	Guid string
}

type Resource struct {
	Metadata Metadata
}

type ListResponse struct {
	TotalResults int `json:"total_results"`
	Resources    []Resource
}

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

func RestartApp(app string) {
	Expect(cf.Cf("restart", app).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
}

func StartApp(app string) {
	Expect(cf.Cf("start", app).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
}

func PushApp(appName, asset, buildpackName string) {
	PushAppNoStart(appName, asset, buildpackName)
	app_helpers.SetBackend(appName)
	StartApp(appName)
}

func GenerateAppName() string {
	return generator.PrefixedRandomName("RATS-APP-")
}

func PushAppNoStart(appName, asset, buildpackName string) {
	Expect(cf.Cf("push", appName,
		"-b", buildpackName,
		"--no-start",
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", asset,
		"-d", config.AppsDomain,
	).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func ScaleAppInstances(appName string, instances int) {
	Expect(cf.Cf("scale", appName, "-i", strconv.Itoa(instances)).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	Eventually(func() string {
		return string(cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT).Out.Contents())
	}, DEFAULT_TIMEOUT*2, 2*time.Second).
		Should(ContainSubstring(fmt.Sprintf("instances: %d/%d", instances, instances)))
}

func DeleteApp(appName string) {
	Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func MapRouteToApp(app, domain, host, path string) {
	Expect(cf.Cf("map-route", app, domain, "--hostname", host, "--path", path).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func DeleteRoute(hostname, contextPath, domain string) {
	Expect(cf.Cf("delete-route", domain,
		"--hostname", hostname,
		"--path", contextPath,
		"-f",
	).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func createRoute(hostname, contextPath, space, domain string) {
	Expect(cf.Cf("create-route", space, domain,
		"--hostname", hostname,
		"--path", contextPath,
	).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func getRouteGuid(hostname, path string) string {
	responseBuffer := cf.Cf("curl", fmt.Sprintf("/v2/routes?q=host:%s&q=path:%s", hostname, path))
	Expect(responseBuffer.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	routeBytes := responseBuffer.Out.Contents()

	var routeResponse ListResponse

	err := json.Unmarshal(routeBytes, &routeResponse)
	Expect(err).NotTo(HaveOccurred())
	Expect(routeResponse.TotalResults).To(Equal(1))

	return routeResponse.Resources[0].Metadata.Guid
}

func GetAppInfo(appName string) (host, port string) {
	var appsResponse AppsResponse
	var statsResponse StatsResponse

	cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
	err := json.Unmarshal(cfResponse, &appsResponse)
	Expect(err).NotTo(HaveOccurred())
	serverAppUrl := appsResponse.Resources[0].Metadata.Url

	cfResponse = cf.Cf("curl", fmt.Sprintf("%s/stats", serverAppUrl)).Wait(DEFAULT_TIMEOUT).Out.Contents()
	err = json.Unmarshal(cfResponse, &statsResponse)
	Expect(err).NotTo(HaveOccurred())

	appIp := statsResponse["0"].Stats.Host
	appPort := fmt.Sprintf("%d", statsResponse["0"].Stats.Port)
	return appIp, appPort
}

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	config = helpers.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	componentName := "Routing"

	rs := []Reporter{}

	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
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
