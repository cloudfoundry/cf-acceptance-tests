package security_groups_test

import (
	"testing"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

var (
	DEFAULT_TIMEOUT   = 30 * time.Second
	CF_PUSH_TIMEOUT   = 2 * time.Minute
	LONG_CURL_TIMEOUT = 2 * time.Minute
)

var (
	context              workflowhelpers.SuiteContext
	config               helpers.Config
	DEFAULT_MEMORY_LIMIT = "256M"
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

	if config.LongCurlTimeout > 0 {
		LONG_CURL_TIMEOUT = config.LongCurlTimeout * time.Second
	}

	context = workflowhelpers.NewContext(config)
	environment := workflowhelpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "SecurityGroups"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func SecurityGroupsDescribe(description string, callback func()) bool {
	BeforeEach(func() {
		if !config.IncludeSecurityGroups {
			Skip(skip_messages.SkipSecurityGroupsMessage)
		}
	})
	return Describe("[security_groups] "+description, callback)
}
