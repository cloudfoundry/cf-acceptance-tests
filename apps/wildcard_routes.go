package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("Wildcard Routes", func() {
	var appNameCatnip string
	var appNameSimple string
	var domainName string
	var orgName string

	curlRoute := func(hostName string, path string) string {
		uri := Config.Protocol() + hostName + "." + domainName + path
		curlCmd := helpers.CurlSkipSSL(true, uri).Wait()
		Expect(curlCmd).To(Exit(0))

		Expect(string(curlCmd.Err.Contents())).To(HaveLen(0))
		return string(curlCmd.Out.Contents())
	}

	BeforeEach(func() {
		orgName = TestSetup.RegularUserContext().Org

		domainName = random_name.CATSRandomName("DOMAIN") + "." + Config.GetAppsDomain()
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("create-shared-domain", domainName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		appNameCatnip = random_name.CATSRandomName("APP")
		appNameSimple = random_name.CATSRandomName("APP")

		Expect(cf.Cf(
			"push", appNameCatnip,
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Catnip,
			"-c", "./catnip",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		Expect(cf.Cf(
			"push", appNameSimple,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameCatnip)
		app_helpers.AppReport(appNameSimple)

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("target", "-o", orgName).Wait()).To(Exit(0))
			Expect(cf.Cf("delete-shared-domain", domainName, "-f").Wait()).To(Exit(0))
		})

		Expect(cf.Cf("delete", appNameCatnip, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", appNameSimple, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("Adding a wildcard route to a domain", func() {
		It("completes successfully", func() {
			wildCardRoute := "*"
			regularRoute := "bar"

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName).Wait()).To(Exit(0))
				Expect(cf.Cf("create-route", domainName, "-n", wildCardRoute).Wait()).To(Exit(0))
			})
			Expect(cf.Cf("create-route", domainName, "-n", regularRoute).Wait()).To(Exit(0))

			Expect(cf.Cf("map-route", appNameCatnip, domainName, "-n", wildCardRoute).Wait()).To(Exit(0))
			Expect(cf.Cf("map-route", appNameSimple, domainName, "-n", regularRoute).Wait()).To(Exit(0))

			Eventually(func() string {
				return curlRoute(regularRoute, "/")
			}).Should(ContainSubstring("Hello"))

			Eventually(func() string {
				return curlRoute("foo", "/")
			}).Should(ContainSubstring("Catnip?"))

			Eventually(func() string {
				return curlRoute("foo.baz", "/")
			}).Should(ContainSubstring("Catnip?"))
		})
	})
})
