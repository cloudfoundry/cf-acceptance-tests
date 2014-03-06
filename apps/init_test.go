package apps

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/config"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
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

	Expect(Cf("api", RegularUserContext.ApiUrl)).To(ExitWith(0))

	Expect(Cf("login",
		"-u", RegularUserContext.Username,
		"-p", RegularUserContext.Password)).To(ExitWith(0))

	Expect(Cf("target",
		"-o", RegularUserContext.Org,
		"-s", RegularUserContext.Space)).To(ExitWith(0))
}

func RemoveHomeConfig() {
	os.RemoveAll(homePath)
}
