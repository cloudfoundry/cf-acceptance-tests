package services_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cats_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var (
	testSetup *workflowhelpers.ReproducibleTestSuiteSetup
	config    cats_config.Config
)

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config = cats_config.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	if config.BrokerStartTimeout > 0 {
		BROKER_START_TIMEOUT = config.BrokerStartTimeout * time.Second
	}

	if config.AsyncServiceOperationTimeout > 0 {
		ASYNC_SERVICE_OPERATION_TIMEOUT = config.AsyncServiceOperationTimeout * time.Second
	}

	testSetup = workflowhelpers.NewTestSuiteSetup(config)

	BeforeSuite(func() {
		testSetup.Setup()
	})

	AfterSuite(func() {
		testSetup.Teardown()
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
			Skip(skip_messages.SkipServicesMessage)
		}
	})
	return Describe("[services] "+description, callback)
}
