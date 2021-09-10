package cats_suite_helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

const (
	CF_JAVA_TIMEOUT              = 10 * time.Minute
	V3_PROCESS_TIMEOUT           = 45 * time.Second
	DEFAULT_MEMORY_LIMIT         = "256M"
	DEFAULT_WINDOWS_MEMORY_LIMIT = "1G"
)

var (
	Config    CatsConfig
	TestSetup *workflowhelpers.ReproducibleTestSuiteSetup
	ScpPath   string
	SftpPath  string
)

func SkipOnK8s(reason string) {
	BeforeEach(func() {
		if Config.RunningOnK8s() {
			Skip(fmt.Sprintf(skip_messages.SkipK8sMessage, reason))
		}
	})
}

func AppSyslogTcpDescribe(description string, callback func()) bool {
	return Describe("[app_syslog_tcp]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeAppSyslogTcp() {
				Skip(skip_messages.SkipAppSyslogTcpMessage)
			}
		})
		Describe(description, callback)
	})
}

func AppsDescribe(description string, callback func()) bool {
	return Describe("[apps]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeApps() {
				Skip(skip_messages.SkipAppsMessage)
			}
		})
		Describe(description, callback)
	})
}

func IsolationSegmentsDescribe(description string, callback func()) bool {
	return Describe("[isolation_segments]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeIsolationSegments() {
				Skip(skip_messages.SkipIsolationSegmentsMessage)
			}
		})
		Describe(description, callback)
	})
}

func DetectDescribe(description string, callback func()) bool {
	return Describe("[detect]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeDetect() {
				Skip(skip_messages.SkipDetectMessage)
			}
		})
		Describe(description, callback)
	})
}

func DockerDescribe(description string, callback func()) bool {
	return Describe("[docker]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeDocker() {
				Skip(skip_messages.SkipDockerMessage)
			}
		})
		Describe(description, callback)
	})
}

func InternetDependentDescribe(description string, callback func()) bool {
	return Describe("[internet_dependent]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeInternetDependent() {
				Skip(skip_messages.SkipInternetDependentMessage)
			}
		})
		Describe(description, callback)
	})
}

func RouteServicesDescribe(description string, callback func()) bool {
	return Describe("[route_services]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeRouteServices() {
				Skip(skip_messages.SkipRouteServicesMessage)
			}
		})
		Describe(description, callback)
	})
}

func RoutingDescribe(description string, callback func()) bool {
	return Describe("[routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeRouting() {
				Skip(skip_messages.SkipRoutingMessage)
			}
		})
		Describe(description, callback)
	})
}

func HTTP2RoutingDescribe(description string, callback func()) bool {
	return Describe("[HTTP/2 routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeHTTP2Routing() {
				Skip(skip_messages.SkipHTTP2RoutingMessage)
			}
		})
		Describe(description, callback)
	})
}

func TCPRoutingDescribe(description string, callback func()) bool {
	return Describe("[tcp routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeTCPRouting() {
				Skip(skip_messages.SkipTCPRoutingMessage)
			}
		})
		Describe(description, callback)
	})
}

func RoutingIsolationSegmentsDescribe(description string, callback func()) bool {
	return Describe("[routing_isolation_segments]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeRoutingIsolationSegments() {
				Skip(skip_messages.SkipRoutingIsolationSegmentsMessage)
			}
		})
		Describe(description, callback)
	})
}

func ZipkinDescribe(description string, callback func()) bool {
	return Describe("[routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeRouting() {
				Skip(skip_messages.SkipRoutingMessage)
			}

			if !Config.GetIncludeZipkin() {
				Skip(skip_messages.SkipZipkinMessage)
			}
		})
		Describe(description, callback)
	})
}

func SecurityGroupsDescribe(description string, callback func()) bool {
	return Describe("[security_groups]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeSecurityGroups() {
				Skip(skip_messages.SkipSecurityGroupsMessage)
			}
		})
		Describe(description, callback)
	})
}

func ServiceDiscoveryDescribe(description string, callback func()) bool {
	return Describe("[service discovery]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeServiceDiscovery() {
				Skip(skip_messages.SkipServiceDiscoveryMessage)
			}
		})
		Describe(description, callback)
	})
}

func ServicesDescribe(description string, callback func()) bool {
	return Describe("[services]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeServices() {
				Skip(skip_messages.SkipServicesMessage)
			}
		})
		Describe(description, callback)
	})
}

func ServiceInstanceSharingDescribe(description string, callback func()) bool {
	return Describe("[service instance sharing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeServiceInstanceSharing() {
				Skip(skip_messages.SkipServiceInstanceSharingMessage)
			}
		})
		Describe(description, callback)
	})
}

func SshDescribe(description string, callback func()) bool {
	return Describe("[ssh]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeSsh() {
				Skip(skip_messages.SkipSSHMessage)
			}
		})
		Describe(description, callback)
	})
}

func V3Describe(description string, callback func()) bool {
	return Describe("[v3]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeV3() {
				Skip(skip_messages.SkipV3Message)
			}
		})
		Describe(description, callback)
	})
}

func TasksDescribe(description string, callback func()) bool {
	return Describe("[tasks]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeTasks() {
				Skip(skip_messages.SkipTasksMessage)
			}
		})
		Describe(description, callback)
	})
}

func GuidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait()).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func CredhubDescribe(description string, callback func()) bool {
	return Describe("[credhub]", func() {
		BeforeEach(func() {
			if !(Config.GetIncludeCredhubAssisted() || Config.GetIncludeCredhubNonAssisted()) {
				Skip(skip_messages.SkipCredhubMessage)
			}
		})
		SkipOnK8s("Credhub not supported")
		Describe(description, callback)
	})
}

func AssistedCredhubDescribe(description string, callback func()) bool {
	return Describe("[assisted credhub]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeCredhubAssisted() {
				Skip(skip_messages.SkipAssistedCredhubMessage)
			}
		})
		Describe(description, callback)
	})
}

func NonAssistedCredhubDescribe(description string, callback func()) bool {
	return Describe("[non-assisted credhub]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeCredhubNonAssisted() {
				Skip(skip_messages.SkipNonAssistedCredhubMessage)
			}
		})
		Describe(description, callback)
	})
}

func WindowsCredhubDescribe(description string, callback func()) bool {
	return Describe("[windows credhub]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeWindows() {
				Skip(skip_messages.SkipWindowsMessage)
			}
			if !(Config.GetIncludeCredhubAssisted() || Config.GetIncludeCredhubNonAssisted()) {
				Skip(skip_messages.SkipCredhubMessage)
			}
		})
		SkipOnK8s("Windows not supported")
		Describe(description, callback)
	})
}

func WindowsAssistedCredhubDescribe(description string, callback func()) bool {
	return Describe("[windows assisted credhub]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeCredhubAssisted() {
				Skip(skip_messages.SkipAssistedCredhubMessage)
			}
		})
		Describe(description, callback)
	})
}

func WindowsNonAssistedCredhubDescribe(description string, callback func()) bool {
	return Describe("[windows non-assisted credhub]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeCredhubNonAssisted() {
				Skip(skip_messages.SkipNonAssistedCredhubMessage)
			}
		})
		Describe(description, callback)
	})
}

func WindowsDescribe(description string, callback func()) bool {
	return Describe("[windows]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeWindows() {
				Skip(skip_messages.SkipWindowsMessage)
			}
		})
		SkipOnK8s("Windows not supported")
		Describe(description, callback)
	})
}

func VolumeServicesDescribe(description string, callback func()) bool {
	return Describe("[volume_services]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeVolumeServices() {
				Skip(skip_messages.SkipVolumeServicesMessage)
			}
			if Config.GetIncludeDocker() {
				Skip(skip_messages.SkipVolumeServicesDockerEnabledMessage)
			}
		})
		Describe(description, callback)
	})
}
