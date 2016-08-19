package detect

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

var (
	CF_JAVA_TIMEOUT      = 10 * time.Minute
	DEFAULT_TIMEOUT      = 30 * time.Second
	DETECT_TIMEOUT       = 5 * time.Minute
	DEFAULT_MEMORY_LIMIT = "256M"
)

var (
	context workflowhelpers.SuiteContext
	config  helpers.Config
)

func TestDetect(t *testing.T) {
	RegisterFailHandler(Fail)

	config = helpers.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.DetectTimeout > 0 {
		DETECT_TIMEOUT = config.DetectTimeout * time.Second
	}

	context = workflowhelpers.NewContext(config)
	environment := workflowhelpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "Buildpack Detection"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func DetectDescribe(description string, callback func()) bool {
	BeforeEach(func() {
		config = helpers.LoadConfig()
		if !config.IncludeDetect {
			Skip(`Skipping this test because config.IncludeDetect is set to false.`)
		}
	})
	return Describe("[detect]"+description, callback)
}
