package ipv6

import (
    "fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
    "github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
    "github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
    "github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = IPv6Describe("IPv6", func() {
    var appName string

    BeforeEach(func() {
        appName = random_name.CATSRandomName("APP")
    })

    AfterEach(func() {
        app_helpers.AppReport(appName)
        Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
    })

    Describe("egress from a Python app", func() {
        for _, stack := range Config.GetStacks() {
            stack := stack
            Context(fmt.Sprintf("when using %s stack", stack), func() {
                It("allows IPv6 egress", func() {
                    Expect(cf.Cf("push", appName,
                        "-m", DEFAULT_MEMORY_LIMIT,
                        "-p", assets.NewAssets().Python,
                        "-s", stack,
                    ).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

                    /* This test is verifying that the default path
                    for testing python buldpack is working */
                    Eventually(func() string {
                        return helpers.CurlApp(Config, appName, "/")
                    }).Should(ContainSubstring("Hello"))

                    /* This test is checking IPv6 egress calls.
                    It examines that after making a request to a predifined route,
                    IPv6 tests are executed successfully */
                    
                    Eventually(func() string {
                        return helpers.CurlApp(Config, appName, "/ipv6-test")
                    }).Should(ContainSubstring("IPv6 tests executed"))
                })
            })
        }
    })
})