package services_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
)

var (
	context workflowhelpers.SuiteContext
	config  helpers.Config
)

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config = helpers.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	if config.BrokerStartTimeout > 0 {
		BROKER_START_TIMEOUT = config.BrokerStartTimeout * time.Second
	}

	context = workflowhelpers.NewContext(config)
	environment := workflowhelpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "Services"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func ServicesDescribe(description string, callback func()) bool {
	BeforeEach(func() {
		if !config.IncludeServices {
			Skip(`Skipping this test because config.IncludeServices is set to false.`)
		}
	})
	return Describe("[services]"+description, callback)
}
