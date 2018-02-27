package service_discovery

import (
	"encoding/json"

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

var _ = ServiceDiscoveryDescribe("Service Discovery", func() {
	var appNameFrontend string
	var appNameBackend string
	var domainName string
	var orgName string
	var spaceName string
	var internalDomainName string
	var internalHostName string

	BeforeEach(func() {
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space
		domainName = random_name.CATSRandomName("DOMAIN") + "." + Config.GetAppsDomain()
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("create-shared-domain", domainName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		internalDomainName = "apps.internal"
		internalHostName = random_name.CATSRandomName("HOST")
		appNameFrontend = random_name.CATSRandomName("APP-FRONT")
		appNameBackend = random_name.CATSRandomName("APP-BACK")

		// check that the internal domain has been seeded
		sharedDomainBody := cf.Cf("curl", "/v2/shared_domains?q=name:apps.internal").Wait(Config.CfPushTimeoutDuration()).Out.Contents()
		var sharedDomainJSON struct {
			Resources []struct {
				Metadata struct {
					SharedDomainGuid string `json:"guid"`
				} `json:"metadata"`
			} `json:"resources"`
		}
		Expect(json.Unmarshal([]byte(sharedDomainBody), &sharedDomainJSON)).To(Succeed())
		Expect(sharedDomainJSON.Resources[0].Metadata.SharedDomainGuid).ToNot(BeNil())

		// push backend app
		Expect(cf.Cf(
			"push", appNameBackend,
			"--no-start",
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
			"-d", Config.GetAppsDomain(),
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		app_helpers.SetBackend(appNameBackend)
		Expect(cf.Cf("start", appNameBackend).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// map internal route to backend app
		Expect(cf.Cf("map-route", appNameBackend, internalDomainName, "--hostname", internalHostName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// push frontend app
		Expect(cf.Cf(
			"push", appNameFrontend,
			"--no-start",
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Proxy,
			"-d", Config.GetAppsDomain(),
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		app_helpers.SetBackend(appNameFrontend)
		Expect(cf.Cf("start", appNameFrontend).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameFrontend, Config.DefaultTimeoutDuration())
		app_helpers.AppReport(appNameBackend, Config.DefaultTimeoutDuration())

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("target", "-o", orgName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("delete-shared-domain", domainName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})

		Expect(cf.Cf("delete", appNameFrontend, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete", appNameBackend, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("Adding an internal route on an app", func() {
		It("successfully creates a policy", func() {
			curlArgs := appNameFrontend + "." + Config.GetAppsDomain() + "/proxy/" + internalHostName + "." + internalDomainName + ":8080"
			Eventually(func() string {
				curl := helpers.Curl(Config, curlArgs).Wait(Config.DefaultTimeoutDuration())
				return string(curl.Out.Contents())
			}, Config.DefaultTimeoutDuration()).ShouldNot(ContainSubstring("Hello, world!"))

			// add a policy
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).ToNot(ContainSubstring(appNameBackend))
				Expect(cf.Cf("add-network-policy", appNameFrontend, "--destination-app", appNameBackend, "--protocol", "tcp", "--port", "8080").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(ContainSubstring(appNameBackend))
			})

			Eventually(func() string {
				curl := helpers.Curl(Config, curlArgs).Wait(Config.DefaultTimeoutDuration())
				return string(curl.Out.Contents())
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, world!"))
		})
	})
})
