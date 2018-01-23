package nimbus

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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = NimbusDescribe("nfs service", func() {

	var appName, serviceName, orgName string

	BeforeEach(func() {

		if Config.GetBackend() != "diego" {
			Skip(skip_messages.SkipDiegoMessage)
		}

		if Config.GetIncludeNimbusServiceNFS() != true {
			Skip("include_nimbus_service_nfs was not set to true")
		}

		appName = random_name.CATSRandomName("APP")
		serviceName = random_name.CATSRandomName("SVC")
		orgName = TestSetup.RegularUserContext().Org
		shareConfig := "{\"share\": \"" + Config.GetNimbusServiceNFSShare() + "\"}"

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf(
				"enable-service-access",
				Config.GetNimbusServiceNameNFS(),
				"-p", Config.GetNimbusServicePlanNFS(),
				"-o", orgName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})

		workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
			Expect(cf.Cf(
				"create-service",
				Config.GetNimbusServiceNameNFS(),
				Config.GetNimbusServicePlanNFS(),
				serviceName,
				"-c", shareConfig).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf(
				"push",
				appName,
				"-p", assets.NewAssets().Pora,
				"-b", Config.GetGoBuildpackName(),
				"-i", "2",
				"--no-start").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf(
				"set-env",
				appName,
				"GOPACKAGENAME",
				"pora").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf(
				"bind-service",
				appName,
				serviceName,
				"-c", "{\"uid\":\"1000\",\"gid\":\"1000\"}").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			app_helpers.EnableDiego(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

	})

	AfterEach(func() {
		Expect(cf.Cf(
			"delete",
			appName,
			"-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		Expect(cf.Cf(
			"delete-service",
			serviceName,
			"-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("can write to nfs server", func() {

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/write")
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello Persistent World!"))

	})

})
