package routing

import (
	"time"

	cats_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	DEFAULT_TIMEOUT   = 1 * time.Minute
	CF_PUSH_TIMEOUT   = 2 * time.Minute
	APP_START_TIMEOUT = 2 * time.Minute

	context workflowhelpers.SuiteContext
	config  cats_config.Config
)

func TestRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	config = cats_config.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	componentName := "Routing"

	rs := []Reporter{}

	context = workflowhelpers.NewContext(config)
	environment := workflowhelpers.NewEnvironment(context)

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

func RoutingDescribe(description string, callback func()) bool {
	BeforeEach(func() {
		if !config.IncludeRouting {
			Skip(skip_messages.SkipRoutingMessage)
		}
	})
	return Describe("[routing] "+description, callback)
}
