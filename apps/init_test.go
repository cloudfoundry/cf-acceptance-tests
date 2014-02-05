package apps

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"

	"github.com/pivotal-cf-experimental/cf-acceptance-tests/config"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

func TestLifecycle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Application Lifecycle", []Reporter{reporters.NewJUnitReporter(fmt.Sprintf("junit_%d.xml", ginkgoconfig.GinkgoConfig.ParallelNode))})
}

var IntegrationConfig = config.Load()

var AppName = ""

var doraPath = "../assets/dora"
var helloPath = "../assets/hello-world"

func AppUri(endpoint string) string {
	return "http://" + AppName + "." + IntegrationConfig.AppsDomain + endpoint
}

func Curling(endpoint string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(AppUri(endpoint))
	}
}
