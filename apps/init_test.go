package apps

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)

	AsUser(RegularUserContext, func () {
		RunSpecsWithDefaultAndCustomReporters(t, "Application Lifecycle", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
	})

}

var IntegrationConfig = LoadConfig()
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
