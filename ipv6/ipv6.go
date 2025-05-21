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

	describeIPv6Tests := func(buildpackPath string, stack string) {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", buildpackPath,
			"-s", stack,
		).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

		for _, data := range EndpointTypeMap {
			response := helpers.CurlApp(Config, appName, data.path)
		
			if data.path == "" {
				Expect(response).To(ContainSubstring("Hello"))
			} else {
				Expect(response).To(ContainSubstring(fmt.Sprintf("%s validation resulted in success", data.validationName)))
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

			Context(fmt.Sprintf("Using Golang stack: %s", stack), func() {
				It("validates IPv6 egress for Golang App", func() {
					describeIPv6Tests(assets.NewAssets().Golang, stack)
				})
			})
		}
	})
})
