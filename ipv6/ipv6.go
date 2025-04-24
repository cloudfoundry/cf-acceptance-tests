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

	Describe("Egress Capability in Python App", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("Using stack: %s", stack), func() {
				It("validates IPv6 egress and examines test results", func() {
					Expect(cf.Cf("push", appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Python,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					response := helpers.CurlApp(Config, appName, "/ipv6-test")

					Expect(response).To(ContainSubstring("IPv4 validation resulted in success"))
					Expect(response).To(ContainSubstring("IPv6 validation resulted in success"))
					Expect(response).To(ContainSubstring("Dual stack validation resulted in success"))
					Expect(response).NotTo(ContainSubstring("validation failed"))
					Expect(response).To(ContainSubstring("validation succeeded"))
				})
			})
		}
	})
})
