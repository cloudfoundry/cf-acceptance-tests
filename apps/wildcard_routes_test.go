package apps

import (
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
	var appNameDora string
	var appNameSimple string
	var domainName string
	var orgName string
	var spaceName string

	curlRoute := func(hostName string, path string) string {
		uri := config.Protocol() + hostName + "." + domainName + path
		curlCmd := helpers.CurlSkipSSL(true, uri).Wait(DEFAULT_TIMEOUT)
		Expect(curlCmd).To(Exit(0))

		Expect(string(curlCmd.Err.Contents())).To(HaveLen(0))
		return string(curlCmd.Out.Contents())
	}

	BeforeEach(func() {
		orgName = context.RegularUserContext().Org
		spaceName = context.RegularUserContext().Space

		domainName = random_name.CATSRandomName("DOMAIN") + "." + config.AppsDomain
		workflowhelpers.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("create-shared-domain", domainName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		})

		appNameDora = random_name.CATSRandomName("APP")
		appNameSimple = random_name.CATSRandomName("APP")

		Expect(cf.Cf(
			"push", appNameDora,
			"--no-start",
			"-b", config.RubyBuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Dora,
			"-d", config.AppsDomain,
		).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		app_helpers.SetBackend(appNameDora)
		Expect(cf.Cf("start", appNameDora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Expect(cf.Cf(
			"push", appNameSimple,
			"--no-start",
			"-b", config.RubyBuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
			"-d", config.AppsDomain,
		).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		app_helpers.SetBackend(appNameSimple)
		Expect(cf.Cf("start", appNameSimple).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameDora, DEFAULT_TIMEOUT)
		app_helpers.AppReport(appNameSimple, DEFAULT_TIMEOUT)

		workflowhelpers.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("target", "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("delete-shared-domain", domainName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		Expect(cf.Cf("delete", appNameDora, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("delete", appNameSimple, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("Adding a wildcard route to a domain", func() {
		It("completes successfully", func() {
			wildCardRoute := "*"
			regularRoute := "bar"

			workflowhelpers.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("target", "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("create-route", spaceName, domainName, "-n", wildCardRoute).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
			Expect(cf.Cf("create-route", spaceName, domainName, "-n", regularRoute).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Expect(cf.Cf("map-route", appNameDora, domainName, "-n", wildCardRoute).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("map-route", appNameSimple, domainName, "-n", regularRoute).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return curlRoute(regularRoute, "/")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello"))

			Eventually(func() string {
				return curlRoute("foo", "/")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

			Eventually(func() string {
				return curlRoute("foo.baz", "/")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})
})
