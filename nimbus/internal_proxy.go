package nimbus

import (

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
)


var _ = NimbusDescribe("proxy service", func() {

	var appName, proxyName, orgName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		proxyName = random_name.CATSRandomName("SVC")
		orgName = TestSetup.RegularUserContext().Org

		// as admin enable service access to internal-proxy
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("enable-service-access", "internal-proxy", "-p", "default", "-o", orgName)
			Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})

		// as user create and bind the internal-proxy service
		workflowhelpers.AsUser(TestSetup.RegularUserContext(), TestSetup.ShortTimeout(), func() {
			Expect(cf.Cf("create-service", "internal-proxy", "default", proxyName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("set-schema", proxyName, assets.NewAssets().NimbusServices + "/proxy-schema.txt" ).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().NimbusServices, "--no-start", "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("bind-service", appName, proxyName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete-service", proxyName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("is accessible in hemel and slough datacenters", func() {

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/proxy")
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("OK"))

	})

})
