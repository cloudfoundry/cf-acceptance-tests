package ipv6

import (
	"fmt"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"net"
	"os"
	"strings"
)

func ValidateIP(ipAddress, expectedType string) {
	parsedIP := net.ParseIP(ipAddress)
	Expect(parsedIP).NotTo(BeNil(), "Expected a valid IP address")

	switch expectedType {
	case "IPv4":
		Expect(parsedIP.To4()).NotTo(BeNil(), "Expected an IPv4 address")
	case "IPv6":
		Expect(parsedIP.To4()).To(BeNil(), "Expected an IPv6 address")
	case "Dual stack":
		Expect(parsedIP).NotTo(BeNil(), "Expected either an IPv4 or IPv6 address")
	}
}

var _ = IPv6Describe("IPv6 Connectivity Tests", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	EndpointTypeMap := map[string]struct {
		expectedType string
		path         string
	}{
		"IPv4": {
			expectedType: "IPv4",
			path:         "/ipv4-test",
		},
		"IPv6": {
			expectedType: "IPv6",
			path:         "/ipv6-test",
		},
		"Dual stack": {
			expectedType: "Dual stack",
			path:         "/dual-stack-test",
		},
		"Default": {
			expectedType: "",
			path:         "",
		},
	}

	pushAndValidate := func(commandOptions []string, defaultPathExpectMessage string) {
		pushSession := cf.Cf(commandOptions...)
		Expect(pushSession.Wait(Config.DetectTimeoutDuration())).To(Exit(0))

		for _, data := range EndpointTypeMap {
			response := helpers.CurlAppWithStatusCode(Config, appName, data.path)

			if data.expectedType == "" {
				Expect(response).To(ContainSubstring(defaultPathExpectMessage))
			} else {
				responseParts := strings.Split(response, "\n")
				ipAddress := responseParts[0]
				statusCode := responseParts[1]

				ValidateIP(ipAddress, data.expectedType)
				Expect(statusCode).To(Equal("200"))
			}
		}
	}

	describeIPv6Tests := func(assetPath, stack string) {
		commandOptions := []string{"push", appName, "-s", stack, "-p", assetPath, "-m", DEFAULT_MEMORY_LIMIT}
		pushAndValidate(commandOptions, "Hello")
	}

	describeIPv6JavaSpringTest := func(stack string) {
		Expect(os.Chdir("assets/java-spring")).NotTo(HaveOccurred())
		commandOptions := []string{"push", appName, "-s", stack}
		pushAndValidate(commandOptions, "ok")
	}

	describeIPv6RubyTests := func(assetPath, stack string) {
		commandOptions := []string{"push", appName, "-s", stack, "-p", assetPath, "-m", DEFAULT_MEMORY_LIMIT}
		pushAndValidate(commandOptions, "Healthy")
	}

	Describe("Egress Capability in Apps", func() {
		for _, stack := range Config.GetStacks() {
			Context(fmt.Sprintf("Using Python stack: %s", stack), func() {
				It("validates IPv6 egress for Python App", func() {
					describeIPv6Tests(assets.NewAssets().Python, stack)
				})
			})
			Context(fmt.Sprintf("Using Node stack: %s", stack), func() {
				It("validates IPv6 egress for Node App", func() {
					describeIPv6Tests(assets.NewAssets().Node, stack)
				})
			})

			Context(fmt.Sprintf("Using Go stack: %s", stack), func() {
				It("validates IPv6 egress for Go App", func() {
					describeIPv6Tests(assets.NewAssets().Golang, stack)
				})
			})

			Context(fmt.Sprintf("Using JavaSpring stack: %s", stack), func() {
				It("validates IPv6 egress for JavaSpring App", func() {
					describeIPv6JavaSpringTest(stack)
				})
			})

			Context(fmt.Sprintf("Using Ruby stack: %s", stack), func() {
				It("validates IPv6 egress for Ruby App", func() {
					describeIPv6RubyTests(assets.NewAssets().RubySimple, stack)
				})
			})

			Context(fmt.Sprintf("Using Nginx stack: %s", stack), func() {
				It("validates IPv6 egress for Nginx App", func() {
					describeIPv6Tests(assets.NewAssets().NginxIPv6, stack)
				})
			})

			Context(fmt.Sprintf("Using PHP stack: %s", stack), func() {
				It("validates IPv6 egress for PHP App", func() {
					describeIPv6Tests(assets.NewAssets().Php, stack)
				})
			})
		}
	})
})
