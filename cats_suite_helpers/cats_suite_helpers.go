package cats_suite_helpers

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
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

func IsolatedTCPRoutingDescribe(description string, callback func()) bool {
	return Describe("[isolated tcp routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeTCPIsolationSegments() {
				Skip(skip_messages.SkipIsolatedTCPRoutingMessage)
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

func CNBDescribe(description string, callback func()) bool {
	return Describe("[cnb]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeCNB() {
				Skip(skip_messages.SkipCNBMessage)
			}
		})
		Describe(description, callback)
	})
}

const (
	BuildpackLifecycle string = "buildpack"
	CNBLifecycle              = "CNB"
	DockerLifecycle           = "Docker"
	WindowsLifecycle          = "windows"
)

func FileBasedServiceBindingsDescribe(description string, lifecycle string, callback func()) bool {
	return Describe(fmt.Sprintf("[file-based service bindings %s]", lifecycle), func() {
		BeforeEach(func() {
			if lifecycle == BuildpackLifecycle && !Config.GetIncludeFileBasedServiceBindings() {
				Skip(skip_messages.SkipFileBasedServiceBindingsBuildpackApp)
			}
			if lifecycle == CNBLifecycle && (!Config.GetIncludeFileBasedServiceBindings() || !Config.GetIncludeCNB()) {
				Skip(skip_messages.SkipFileBasedServiceBindingsCnbApp)
			}
			if lifecycle == DockerLifecycle && (!Config.GetIncludeFileBasedServiceBindings() || !Config.GetIncludeDocker()) {
				Skip(skip_messages.SkipFileBasedServiceBindingsDockerApp)
			}
			if lifecycle == WindowsLifecycle && (!Config.GetIncludeFileBasedServiceBindings() || !Config.GetIncludeWindows()) {
				Skip(skip_messages.SkipFileBasedServiceBindingsWindowsApp)
			}
		})
		Describe(description, callback)
	})
}

func IPv6Describe(description string, callback func()) bool {
	return Describe("[ipv6]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeIPv6() {
				Skip(skip_messages.SkipIPv6)
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

func TCPSNIRoutingDescribe(description string, callback func()) bool {
	return Describe("[tcp sni routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeTCPSNIRouting() {
				Skip(skip_messages.SkipTCPSNIRoutingMessage)
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

func CommaDelimitedSecurityGroupsDescribe(description string, callback func()) bool {
	return Describe("[comma_delimited_security_groups]", func() {
		BeforeEach(func() {
			if !Config.GetCommaDelimitedASGsEnabled() {
				Skip(skip_messages.SkipCommaDelimitedSecurityGroupsMessage)
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

func ServiceCredentialBindingRotationDescribe(description string, callback func()) bool {
	return Describe("[service credential binding rotation]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeServiceCredentialBindingRotation() {
				Skip(skip_messages.SkipServiceCredentialBindingRotationMessage)
			}
		})
		Describe(description, callback)
	})
}

func ServicesSsoDescribe(description string, callback func()) bool {
	return Describe("[services sso]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeSSO() {
				Skip(skip_messages.SkipSSOMessage)
			}
		})
		Describe(description, callback)
	})
}

func UserProvidedServicesDescribe(description string, callback func()) bool {
	return Describe("[user provided services]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeUserProvidedServices() {
				Skip(skip_messages.SkipUserProvidedServicesMessage)
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
		Describe(description, callback)
	})
}

func WindowsTCPRoutingDescribe(description string, callback func()) bool {
	return Describe("[windows routing]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeTCPRouting() || !Config.GetIncludeWindows() {
				Skip(skip_messages.SkipTCPRoutingMessage)
			}
		})
		Describe(description, callback)
	})
}

func VolumeServicesDescribe(description string, callback func()) bool {
	return Describe("[volume_services]", func() {
		BeforeEach(func() {
			if !Config.GetIncludeVolumeServices() {
				Skip(skip_messages.SkipVolumeServicesMessage)
			}
		})
		Describe(description, callback)
	})
}

func GetNServerResponses(n int, domainName, externalPort1 string) ([]string, error) {
	var responses []string

	for i := 0; i < n; i++ {
		resp, err := SendAndReceive(domainName, externalPort1)
		if err != nil {
			return nil, err
		}

		responses = append(responses, resp)
	}

	return responses, nil
}

func GetTCPPort() int {
	start := Config.GetTCPPortRangeStart()
	end := Config.GetTCPPortRangeEnd()
	if start > 0 && end >= start {
		return rand.Intn(end-start+1) + start
	}
	return 0
}

func MapTCPRoute(appName, domainName string) string {
	port := GetTCPPort()
	args := []string{"map-route", appName, domainName}
	if port != 0 {
		args = append(args, "--port", strconv.Itoa(port))
	}
	createRouteSession := cf.Cf(args...).Wait()
	Expect(createRouteSession).To(Exit(0))

	r := regexp.MustCompile(fmt.Sprintf(`.+%s:(\d+).+`, domainName))
	return r.FindStringSubmatch(string(createRouteSession.Out.Contents()))[1]
}

func SendAndReceive(addr string, externalPort string) (string, error) {
	address := fmt.Sprintf("%s:%s", addr, externalPort)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	message := []byte(fmt.Sprintf("Time is %d", time.Now().Nanosecond()))

	_, err = conn.Write(message)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok {
			if ne.Temporary() {
				return SendAndReceive(addr, externalPort)
			}
		}

		return "", err
	}

	// see https://github.com/cloudfoundry/cf-acceptance-tests/issues/1173
	time.Sleep(100 * time.Millisecond)

	buff := make([]byte, 1024)
	_, err = conn.Read(buff)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok {
			if ne.Temporary() {
				return SendAndReceive(addr, externalPort)
			}
		}

		return "", err
	}

	// only grab up to the first null byte of a message since we have a predefined slice length that may not be full
	i := len(buff)

	if j := bytes.IndexByte(buff, 0); j > 0 {
		i = j
	}

	return string(buff[:i]), nil
}

// MapTCPRouteWithHostname maps a TCP route on `domainName` with the given hostname.
// If port == 0, a new external port is allocated by the TCP router; the chosen port
// is parsed from the CLI output and returned.
// If port != 0, the supplied port is reused so callers can map multiple apps on the
// same external port, differentiated by SNI hostname.
func MapTCPRouteWithHostname(appName, domainName, hostname string, port int) int {
	if port == 0 {
		port = GetTCPPort()
	}
	args := []string{"map-route", appName, domainName, "--hostname", hostname}
	if port != 0 {
		args = append(args, "--port", strconv.Itoa(port))
	}

	session := cf.Cf(args...).Wait()
	Expect(session).To(Exit(0))

	if port != 0 {
		return port
	}

	r := regexp.MustCompile(fmt.Sprintf(`.+%s:(\d+).+`, domainName))
	matches := r.FindStringSubmatch(string(session.Out.Contents()))
	Expect(matches).To(HaveLen(2), "expected map-route output to contain an external port")

	chosen, err := strconv.Atoi(matches[1])
	Expect(err).ToNot(HaveOccurred())

	return chosen
}

// SendAndReceiveTLS opens a TLS connection to addr:port using the given SNI serverName,
// writes a small payload, and returns the server's response. insecureSkipVerify should
// typically be true in CATs since we only validate routing, not cert trust.
func SendAndReceiveTLS(addr string, port int, serverName string, insecureSkipVerify bool) (string, error) {
	address := fmt.Sprintf("%s:%d", addr, port)

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: insecureSkipVerify,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", err
	}

	message := []byte(fmt.Sprintf("Time is %d", time.Now().Nanosecond()))
	if _, err = conn.Write(message); err != nil {
		return "", err
	}

	// mirror SendAndReceive: small pause to let the server respond before the read.
	time.Sleep(100 * time.Millisecond)

	buff := make([]byte, 1024)
	n, err := conn.Read(buff)
	if err != nil {
		return "", err
	}

	i := n
	if j := bytes.IndexByte(buff[:n], 0); j > 0 {
		i = j
	}

	return string(buff[:i]), nil
}

// GetNTLSResponses performs n TLS SNI round-trips against addr:port using the given
// serverName and returns all responses in order.
func GetNTLSResponses(n int, addr string, port int, serverName string, insecureSkipVerify bool) ([]string, error) {
	responses := make([]string, 0, n)
	for i := 0; i < n; i++ {
		resp, err := SendAndReceiveTLS(addr, port, serverName, insecureSkipVerify)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}
	return responses, nil
}
