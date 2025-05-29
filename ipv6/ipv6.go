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

			Context(fmt.Sprintf("Using Ruby stack: %s", stack), func() {
				It("validates IPv6 egress for Ruby App", func() {
					describeIPv6RubyTests(assets.NewAssets().RubySimple, stack)
				})
			})
		}
	})
})
