package service_discovery

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)

const defaultInternalDomain = "apps.internal"

var _ = ServiceDiscoveryDescribe("Service Discovery", func() {
	var appNameFrontend string
	var appNameBackend string
	var internalHostName string
	var orgName string
	var spaceName string

	BeforeEach(func() {
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space

		internalHostName = random_name.CATSRandomName("HOST")
		appNameFrontend = random_name.CATSRandomName("APP-FRONT")
		appNameBackend = random_name.CATSRandomName("APP-BACK")

		// push backend app
		Expect(cf.Cf(
			"push", appNameBackend,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// map internal route to backend app
		Expect(cf.Cf("map-route", appNameBackend, defaultInternalDomain, "--hostname", internalHostName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// push frontend app
		Expect(cf.Cf(
			"push", appNameFrontend,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Proxy,
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameFrontend)
		app_helpers.AppReport(appNameBackend)

		Expect(cf.Cf("delete", appNameFrontend, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", appNameBackend, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("Adding an internal route on an app", func() {
		It("successfully creates a policy", func() {
			curlArgs := Config.Protocol() + appNameFrontend + "." + Config.GetAppsDomain() + "/proxy/" + internalHostName + "." + defaultInternalDomain + ":8080"
			Eventually(func() string {
				curl := helpers.Curl(Config, curlArgs).Wait()
				return string(curl.Out.Contents())
			}).ShouldNot(ContainSubstring("Hello, world!"))

			// add a policy
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).ToNot(ContainSubstring(appNameBackend))
				Expect(cf.Cf("add-network-policy", appNameFrontend, "--destination-app", appNameBackend, "--protocol", "tcp", "--port", "8080").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).To(ContainSubstring(appNameBackend))
			})

			Eventually(func() string {
				curl := helpers.Curl(Config, curlArgs).Wait()
				return string(curl.Out.Contents())
			}).Should(ContainSubstring("Hello, world!"))
		})
	})
})
