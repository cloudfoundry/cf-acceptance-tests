package services_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
)

var _ = ServicesDescribe("Purging service instances", func() {
	var broker ServiceBroker
	var appName, instanceName string

	AfterEach(func() {
		app_helpers.AppReport(broker.Name)
		broker.Destroy()
	})

	Context("for public brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			broker.Push(Config)
			broker.Configure()
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				broker.Create()
				broker.PublicizePlans()
			})
			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVIN")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
			Expect(cf.Cf("delete-service", instanceName, "-f").Wait()).To(Exit(0))
		})

		It("removes the service instance", func() {
			By("Having a bound service instance")
			createApp := cf.Cf("push",
				appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "./catnip",
			).Wait(Config.CfPushTimeoutDuration())
			Expect(createApp).To(Exit(0), "failed creating app")

			broker.CreateServiceInstance(instanceName)

			services := cf.Cf("services").Wait()
			Expect(services).To(Exit(0))
			Expect(services).To(Say(instanceName))

			bindService := cf.Cf("bind-service", appName, instanceName).Wait()
			Expect(bindService).To(Exit(0), "failed binding app to service")

			By("Making the broker unavailable")
			Expect(cf.Cf("delete", broker.Name, "-f", "-r").Wait()).To(Exit(0))

			By("Purging the service instance")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				TestSetup.RegularUserContext().TargetSpace()
				Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait()).To(Exit(0))
			})

			By("Ensuring the service instance is gone")
			services = cf.Cf("services").Wait()
			Expect(services).To(Exit(0))
			Expect(services).NotTo(Say(instanceName))
			Expect(services).NotTo(Say(appName))
		})
	})

	Context("for space scoped brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			TestSetup.RegularUserContext().TargetSpace()
			broker.Push(Config)
			broker.Configure()
			broker.CreateSpaceScoped()
			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVIN")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
			Expect(cf.Cf("delete-service", instanceName, "-f").Wait()).To(Exit(0))
		})

		It("removes the service instance", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), TestSetup.ShortTimeout(), func() {
				By("Having a bound service instance")
				createApp := cf.Cf("push",
					appName,
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-p", assets.NewAssets().Catnip,
					"-c", "./catnip",
				).Wait(Config.CfPushTimeoutDuration())
				Expect(createApp).To(Exit(0), "failed creating app")

				broker.CreateServiceInstance(instanceName)

				services := cf.Cf("services").Wait()
				Expect(services).To(Exit(0))
				Expect(services).To(Say(instanceName))

				bindService := cf.Cf("bind-service", appName, instanceName).Wait()
				Expect(bindService).To(Exit(0), "failed binding app to service")

				By("Making the broker unavailable")
				Expect(cf.Cf("delete", broker.Name, "-f", "-r").Wait()).To(Exit(0))

				By("Purging the service instance")
				Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait()).To(Exit(0))

				By("Ensuring the service instance is gone")
				services = cf.Cf("services").Wait()
				Expect(services).To(Exit(0))
				Expect(services).NotTo(Say(instanceName))
				Expect(services).NotTo(Say(appName))
			})
		})
	})
})
