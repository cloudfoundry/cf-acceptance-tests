package route_services

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
	DEFAULT_TIMEOUT = 30 * time.Second
	CF_PUSH_TIMEOUT = 2 * time.Minute

	testSetup *workflowhelpers.ReproducibleTestSuiteSetup
	config    cats_config.Config
)

func TestRouteServices(t *testing.T) {
	RegisterFailHandler(Fail)

	config = cats_config.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	componentName := "Route Services"

	rs := []Reporter{}

	testSetup = workflowhelpers.NewTestSuiteSetup(config)

	BeforeSuite(func() {
		testSetup.Setup()
	})

	AfterSuite(func() {
		testSetup.Teardown()
	})

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func RouteServicesDescribe(description string, callback func()) bool {
	BeforeEach(func() {
		if !config.IncludeRouteServices {
			Skip(skip_messages.SkipRouteServicesMessage)
		}
	})
	return Describe("[route_services] "+description, callback)
}
