package cats_suite_helpers

import (
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

const (
	APP_START_TIMEOUT    = 2 * time.Minute
	CF_JAVA_TIMEOUT      = 10 * time.Minute
	DEFAULT_MEMORY_LIMIT = "256M"
)

var (
	Config    CatsConfig
	TestSetup *workflowhelpers.ReproducibleTestSuiteSetup
	ScpPath   string
	SftpPath  string
)

func AppsDescribe(description string, callback func()) bool {
	return Describe("[apps] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeApps() {
				Skip(`Skipping this test because Config.IncludeApps is set to 'false'.`)
			}
		})
		callback()
	})
}

func IsolationSegmentsDescribe(description string, callback func()) bool {
	return Describe("[isolation_segments] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeIsolationSegments() {
				Skip(`Skipping this test because Config.IncludeIsolationSegments is set to 'false'.`)
			}
		})
		callback()
	})
}

func BackendCompatibilityDescribe(description string, callback func()) bool {
	return Describe("[backend_compatibility] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeBackendCompatiblity() {
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
			if !Config.GetIncludeDetect() {
				Skip(`Skipping this test because Config.IncludeDetect is set to 'false'.`)
			}
		})
		callback()
	})
}

func DockerDescribe(description string, callback func()) bool {
	return Describe("[docker] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeDocker() {
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
			if !Config.GetIncludeInternetDependent() {
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
			if !Config.GetIncludeRouteServices() {
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
			if !Config.GetIncludeRouting() {
				Skip(`Skipping this test because Config.IncludeRouting is set to 'false'.`)
			}
		})
		callback()
	})
}

func RoutingIsolationSegmentsDescribe(description string, callback func()) bool {
	return Describe("[routing_isolation_segments] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeRoutingIsolationSegments() {
				Skip(`Skipping this test because Config.IncludeRoutingIsolationSegments is set to 'false'.`)
			}
		})
		callback()
	})
}

func ZipkinDescribe(description string, callback func()) bool {
	return Describe("[routing] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeRouting() {
				Skip(`Skipping this test because Config.IncludeRouting is set to 'false'`)
			}

			if !Config.GetIncludeZipkin() {
				Skip(`Skipping this test because Config.IncludeZipkin is set to 'false'`)
			}
		})
		callback()
	})
}

func SecurityGroupsDescribe(description string, callback func()) bool {
	return Describe("[security_groups] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeSecurityGroups() {
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
			if !Config.GetIncludeServices() {
				Skip(`Skipping this test because Config.IncludeServices is set to 'false'.`)
			}
		})
		callback()
	})
}

func SshDescribe(description string, callback func()) bool {
	return Describe("[ssh] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeSsh() {
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
			if !Config.GetIncludeV3() {
				Skip(`Skipping this test because Config.IncludeV3 is set to 'false'.`)
			}
		})
		callback()
	})
}

func TasksDescribe(description string, callback func()) bool {
	return Describe("[tasks] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludeTasks() {
				Skip(`Skipping this test because Config.IncludeTasks is set to 'false'.`)
			}
		})
		callback()
	})
}

func PersistentAppDescribe(description string, callback func()) bool {
	return Describe("[persistent_app] "+description, func() {
		BeforeEach(func() {
			if !Config.GetIncludePersistentApp() {
				Skip(`Skipping this test because Config.IncludePersistentApp is set to 'false'.`)
			}
		})
		callback()
	})
}

func GuidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}
