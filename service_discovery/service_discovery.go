package service_discovery

import (
	"encoding/json"
	"fmt"

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

		internalDomainName = Config.GetInternalDomain()
		internalHostName = random_name.CATSRandomName("HOST")
		appNameFrontend = random_name.CATSRandomName("APP-FRONT")
		appNameBackend = random_name.CATSRandomName("APP-BACK")

		// check that the internal domain exists
		sharedDomainBody := cf.Cf("curl", fmt.Sprintf("/v2/shared_domains?q=name:%s", internalDomainName)).Wait(Config.CfPushTimeoutDuration()).Out.Contents()
		var sharedDomainJSON struct {
			Resources []struct {
				Entity struct {
					Internal bool `json:"internal"`
				} `json:"entity"`
			} `json:"resources"`
		}
		Expect(json.Unmarshal([]byte(sharedDomainBody), &sharedDomainJSON)).To(Succeed())
		Expect(sharedDomainJSON.Resources).ToNot(BeEmpty())
		Expect(sharedDomainJSON.Resources).To(HaveLen(1), fmt.Sprintf("shared domain %q doesn't exist", internalDomainName))
		Expect(sharedDomainJSON.Resources[0].Entity.Internal).To(BeTrue(), fmt.Sprintf("%q not an internal domain", internalDomainName))

		// push backend app
		Expect(cf.Cf(
			"push", appNameBackend,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
			"-d", Config.GetAppsDomain(),
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// map internal route to backend app
		Expect(cf.Cf("map-route", appNameBackend, internalDomainName, "--hostname", internalHostName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		// push frontend app
		Expect(cf.Cf(
			"push", appNameFrontend,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Proxy,
			"-d", Config.GetAppsDomain(),
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appNameFrontend)
		app_helpers.AppReport(appNameBackend)

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf("target", "-o", orgName).Wait()).To(Exit(0))
			Expect(cf.Cf("delete-shared-domain", domainName, "-f").Wait()).To(Exit(0))
		})

		Expect(cf.Cf("delete", appNameFrontend, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", appNameBackend, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("Adding an internal route on an app", func() {
		It("successfully creates a policy", func() {
			curlArgs := appNameFrontend + "." + Config.GetAppsDomain() + "/proxy/" + internalHostName + "." + internalDomainName + ":8080"
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
