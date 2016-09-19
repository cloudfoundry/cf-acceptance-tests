package v3

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	cf_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

var testSetup *workflowhelpers.ReproducibleTestSuiteSetup
var config cf_config.Config

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	config = cf_config.LoadConfig()

	if config.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = config.DefaultTimeout * time.Second
	}

	if config.SleepTimeout > 0 {
		SLEEP_TIMEOUT = config.SleepTimeout * time.Second
	}

	if config.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = config.CfPushTimeout * time.Second
	}

	if config.LongCurlTimeout > 0 {
		LONG_CURL_TIMEOUT = config.LongCurlTimeout * time.Second
	}

	testSetup = workflowhelpers.NewTestSuiteSetup(config)

	BeforeSuite(func() {
		testSetup.Setup()
	})

	AfterSuite(func() {
		testSetup.Teardown()
	})

	componentName := "V3"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func V3Describe(description string, callback func()) bool {
	BeforeEach(func() {
		if !config.IncludeV3 {
			Skip(skip_messages.SkipV3Message)
		}
	})
	return Describe("[v3] "+description, callback)
}
