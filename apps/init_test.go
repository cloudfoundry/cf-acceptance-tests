package apps

import (
	"fmt"
	"testing"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	"github.com/pivotal-cf-experimental/cf-acceptance-tests/config"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)

	CreateHomeConfig()
	RunSpecsWithDefaultAndCustomReporters(t, "Application Lifecycle", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
	RemoveHomeConfig()
}

var IntegrationConfig = config.Load()
var homePath string
var AppName = ""

var doraPath = "../assets/dora"
var helloPath = "../assets/hello-world"
var serviceBrokerPath = "../assets/service_broker"

func AppUri(endpoint string) string {
	return "http://" + AppName + "." + IntegrationConfig.AppsDomain + endpoint
}

func Curling(endpoint string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(AppUri(endpoint))
	}
}

func CreateHomeConfig() {
	homePath = fmt.Sprintf("%s/cf_config_%s", os.Getenv("HOME"), strconv.Itoa(ginkgoconfig.GinkgoConfig.ParallelNode))
	os.MkdirAll(homePath, os.ModePerm)
	os.Setenv("CF_HOME", homePath)

	Expect(Cf("api", os.Getenv("API_ENDPOINT"))).To(ExitWith(0))

	Expect(Cf("login",
		"-u", os.Getenv("CF_USER"),
		"-p", os.Getenv("CF_USER_PASSWORD"))).To(ExitWith(0))

	Expect(Cf("target",
		"-o", os.Getenv("CF_ORG"),
		"-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}

func RemoveHomeConfig() {
	os.RemoveAll(homePath)
}
