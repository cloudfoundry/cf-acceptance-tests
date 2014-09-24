package diego

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

const (
	DEFAULT_TIMEOUT   = 30 * time.Second
	CF_PUSH_TIMEOUT   = 4 * time.Minute
	LONG_CURL_TIMEOUT = 4 * time.Minute

	DIEGO_NULL_BUILDPACK = "https://github.com/cloudfoundry-incubator/null-buildpack/archive/master.zip"
	DEA_NULL_BUILDPACK   = "https://github.com/cloudfoundry-incubator/null-buildpack"
)

var context helpers.SuiteContext

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config := helpers.LoadConfig()
	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "Diego"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}
