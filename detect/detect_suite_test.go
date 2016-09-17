package detect

import (
	"testing"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cf_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
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
	config  cf_config.Config
)

func TestDetect(t *testing.T) {
	RegisterFailHandler(Fail)

	config = cf_config.LoadConfig()

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
		if !config.IncludeDetect {
			Skip(skip_messages.SkipDetectMessage)
		}
	})
	return Describe("[detect] "+description, callback)
}
