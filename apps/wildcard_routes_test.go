package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Wildcard Routes", func() {
	var appNameDora string
	var appNameSimple string
	var domainName string
	var orgName string
	var spaceName string

	curlRoute := func(hostName string, path string) string {
		uri := config.Protocol() + hostName + "." + domainName + path
		curlCmd := runner.Curl(uri)

		runner.NewCmdRunner(curlCmd, DEFAULT_TIMEOUT).Run()
		Expect(string(curlCmd.Err.Contents())).To(HaveLen(0))
		return string(curlCmd.Out.Contents())
	}

	BeforeEach(func() {
		orgName = context.RegularUserContext().Org
		spaceName = context.RegularUserContext().Space

		domainName = generator.RandomName() + "." + helpers.LoadConfig().AppsDomain
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			Expect(cf.Cf("create-shared-domain", domainName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		})

		appNameDora = generator.PrefixedRandomName("CATS-APP-")
		Expect(cf.Cf("push", appNameDora, "-m", "128M", "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		appNameSimple = generator.PrefixedRandomName("CATS-APP-")
		Expect(cf.Cf("push", appNameSimple, "-m", "128M", "-p", assets.NewAssets().HelloWorld, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
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

			cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
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
		})
	})
})
