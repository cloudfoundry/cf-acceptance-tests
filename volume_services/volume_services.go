package volume_services

import (
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = VolumeServicesDescribe("Volume Services", func() {
	var (
		serviceName         string
		serviceInstanceName string
		appName             string
		poraAsset           = assets.NewAssets().Pora
	)

	BeforeEach(func() {
		serviceName = Config.GetVolumeServiceName()
		serviceInstanceName = random_name.CATSRandomName("SVIN")
		appName = random_name.CATSRandomName("APP")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("enable-service-access", serviceName, "-o", TestSetup.RegularUserContext().Org).Wait()
			Expect(session).To(Exit(0))
		})

		By("pushing an app")
		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", poraAsset,
			"-f", filepath.Join(poraAsset, "manifest.yml"),
			"-d", Config.GetAppsDomain(),
			"--no-start",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("creating a service")
		createServiceSession := cf.Cf("create-service", serviceName, Config.GetVolumeServicePlanName(), serviceInstanceName, "-c", Config.GetVolumeServiceCreateConfig())
		Expect(createServiceSession.Wait(TestSetup.ShortTimeout())).To(Exit(0))

		By("binding the service")
		bindSession := cf.Cf("bind-service", appName, serviceInstanceName, "-c", Config.GetVolumeServiceBindConfig())
		Expect(bindSession.Wait(TestSetup.ShortTimeout())).To(Exit(0))

		By("starting the app")
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		Eventually(cf.Cf("delete-service", serviceInstanceName, "-f")).Should(Exit(0))

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			session := cf.Cf("disable-service-access", serviceName, "-o", TestSetup.RegularUserContext().Org).Wait()
			Expect(session).To(Exit(0))
		})
	})

	It("should be able to write to the volume", func() {
		Expect(helpers.CurlApp(Config, appName, "/write")).To(ContainSubstring("Hello Persistent World"))
	})
})
