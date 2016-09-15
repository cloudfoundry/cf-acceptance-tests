package cats_suite_helpers

import (
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var (
	APP_START_TIMEOUT      = 2 * time.Minute
	CF_JAVA_TIMEOUT        = 10 * time.Minute
	CF_PUSH_TIMEOUT        = 2 * time.Minute
	DEFAULT_MEMORY_LIMIT   = "256M"
	DEFAULT_TIMEOUT        = 30 * time.Second
	DETECT_TIMEOUT         = 5 * time.Minute
	LONG_CURL_TIMEOUT      = 2 * time.Minute
	SLEEP_TIMEOUT          = 30 * time.Second
	CF_MARKETPLACE_TIMEOUT = 200 * time.Second
)

var (
	Config      config.Config
	UserContext workflowhelpers.SuiteContext
	ScpPath     string
	SftpPath    string
)

func AppsDescribe(description string, callback func()) bool {
	return Describe("[apps] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeApps {
				Skip(`Skipping this test because Config.IncludeApps is set to 'false'.`)
			}
		})
		callback()
	})
}

func BackendCompatibilityDescribe(description string, callback func()) bool {
	return Describe("[backend_compatibility] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeBackendCompatiblity {
				Skip(`Skipping this test because Config.IncludeBackendCompatibility is set to 'false'.
			NOTE: Ensure that your deployment has deployed both DEA and Diego before running this test.`)
			}
		})
		callback()
	})
}

func DetectDescribe(description string, callback func()) bool {
	return Describe("[detect] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeDetect {
				Skip(`Skipping this test because Config.IncludeDetect is set to 'false'.`)
			}
		})
		callback()
	})
}

func DockerDescribe(description string, callback func()) bool {
	return Describe("[docker] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeDocker {
				Skip(`Skipping this test because Config.IncludeDocker is set to 'false'.
				NOTE: Ensure Docker containers are enabled on your platform before enabling this test.`)
			}
		})
		callback()
	})
}

func TestCliVersionCheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CliVersionCheck Suite")
}

func InternetDependentDescribe(description string, callback func()) bool {
	return Describe("[internet_dependent] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeInternetDependent {
				Skip(`Skipping this test because Config.IncludeInternetDependent is set to 'false'.
NOTE: Ensure that your deployment has access to the internet before running this test.`)
			}
		})
		callback()
	})
}

func RouteServicesDescribe(description string, callback func()) bool {
	return Describe("[route_services] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeRouteServices {
				Skip(`Skipping this test because Config.IncludeRouteServices is set to 'false'.
			NOTE: Ensure that route services are enabled in your deployment before running this test.`)
			}
		})
		callback()
	})
}

func RoutingDescribe(description string, callback func()) bool {
	return Describe("[routing] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeRouting {
				Skip(`Skipping this test because Config.IncludeRouting is set to 'false'.`)
			}
		})
		callback()
	})
}

func SecurityGroupsDescribe(description string, callback func()) bool {
	return Describe("[security_groups] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeSecurityGroups {
				Skip(`Skipping this test because Config.IncludeSecurityGroups is set to 'false'.
			NOTE: Ensure that your deployment restricts internal network traffic by default in order to run this test.`)
			}
		})
		callback()
	})
}

func ServicesDescribe(description string, callback func()) bool {
	return Describe("[services] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeServices {
				Skip(`Skipping this test because Config.IncludeServices is set to 'false'.`)
			}
		})
		callback()
	})
}

func SshDescribe(description string, callback func()) bool {
	return Describe("[ssh] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeSsh {
				Skip(`Skipping this test because Config.IncludeSsh is set to 'false'.
			NOTE: Ensure that your platform is deployed with a Diego SSH proxy in order to run this test.`)
			}
		})
		callback()
	})
}

func V3Describe(description string, callback func()) bool {
	return Describe("[v3] "+description, func() {
		BeforeEach(func() {
			if !Config.IncludeV3 {
				Skip(`Skipping this test because Config.IncludeV3 is set to 'false'.`)
			}
		})
		callback()
	})
}

func GuidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}
