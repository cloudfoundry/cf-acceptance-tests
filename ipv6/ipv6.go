package ipv6

import (
	"encoding/json"
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
)

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
		validationName string
		path           string
	}{
		"api.ipify.org": {
			validationName: "IPv4",
			path:           "/ipv4-test",
		},
		"api6.ipify.org": {
			validationName: "IPv6",
			path:           "/ipv6-test",
		},
		"api64.ipify.org": {
			validationName: "Dual stack",
			path:           "/dual-stack-test",
		},
		"default": {
			validationName: "Default app",
			path:           "",
		},
	}

	pushAndValidate := func(commandOptions []string, defaultPathExpectMessage string) {
		pushSession := cf.Cf(commandOptions...)
		Expect(pushSession.Wait(Config.DetectTimeoutDuration())).To(Exit(0))

		for key, data := range EndpointTypeMap {
			response := helpers.CurlApp(Config, appName, data.path)

			if key == "default" {
				Expect(response).To(ContainSubstring(defaultPathExpectMessage))
			} else {
				Expect(response).To(ContainSubstring(fmt.Sprintf("%s validation resulted in success", data.validationName)))
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

	describeIpv6NginxTest := func(assetPath, stack string) {
		pushSession := cf.Cf("push", appName, "-s", stack, "-p", assetPath, "-m", DEFAULT_MEMORY_LIMIT)
		Expect(pushSession.Wait(Config.DetectTimeoutDuration())).To(Exit(0))

		isIPv4 := func(ip string) bool {
			parsedIP := net.ParseIP(ip)
			return parsedIP != nil && parsedIP.To4() != nil
		}

		isIPv6 := func(ip string) bool {
			parsedIP := net.ParseIP(ip)
			return parsedIP != nil && parsedIP.To4() == nil
		}

		for key, data := range EndpointTypeMap {
			response := helpers.CurlApp(Config, appName, data.path)

			if key == "default" {
				Expect(response).To(ContainSubstring("Hello NGINX!"))
			} else {
				var result map[string]interface{}
				Expect(json.Unmarshal([]byte(response), &result)).To(Succeed())
				ip, ok := result["ip"].(string)
				Expect(ok).To(BeTrue())

				validationResult := false

				switch data.validationName {
				case "IPv4":
					validationResult = isIPv4(ip)
				case "IPv6":
					validationResult = isIPv6(ip)
				case "Dual stack":
					validationResult = isIPv4(ip) || isIPv6(ip)
				}

				Expect(validationResult).To(BeTrue(), fmt.Sprintf("%s validation failed with the following error: %s", data.validationName, ip))
			}
		}
	}

	Describe("Egress Capability in Apps", func() {
		for _, stack := range Config.GetStacks() {

			Context(fmt.Sprintf("Using Python stack: %s", stack), func() {
				It("validates IPv6 egress for Python App", func() {
					describeIPv6Tests(assets.NewAssets().Python, stack)
				})
			})

			Context(fmt.Sprintf("Using Node.js stack: %s", stack), func() {
				It("validates IPv6 egress for Node.js App", func() {
					describeIPv6Tests(assets.NewAssets().Node, stack)
				})
			})

			Context(fmt.Sprintf("Using JavaSpring stack: %s", stack), func() {
				It("validates IPv6 egress for JavaSpring App", func() {
					describeIPv6JavaSpringTest(stack)
				})
			})

			Context(fmt.Sprintf("Using Golang stack: %s", stack), func() {
				It("validates IPv6 egress for Golang App", func() {
					describeIPv6Tests(assets.NewAssets().Golang, stack)
				})
			})

			Context(fmt.Sprintf("Using Nginx stack: %s", stack), func() {
				It("validates IPv6 egress for Nginx App", func() {
					describeIpv6NginxTest(assets.NewAssets().NginxIPv6, stack)
				})
			})
		}
	})
})
