package services_test

import (
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
		app_helpers.AppReport(broker.Name, DEFAULT_TIMEOUT)
		broker.Destroy()
	})

	Context("for public brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BROKER"),
				assets.NewAssets().ServiceBroker,
				testSetup,
			)
			broker.Push(config)
			broker.Configure()
			workflowhelpers.AsUser(testSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				broker.Create()
				broker.PublicizePlans()
			})
			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVCINS")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes the service instance", func() {
			By("Having a bound service instance")
			createApp := cf.Cf("push", appName, "--no-start", "-b", config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)
			Expect(createApp).To(Exit(0), "failed creating app")
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			broker.CreateServiceInstance(instanceName)

			services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
			Expect(services).To(Exit(0))
			Expect(services).To(Say(instanceName))

			bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(bindService).To(Exit(0), "failed binding app to service")

			By("Making the broker unavailable")
			Expect(cf.Cf("delete", broker.Name, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			By("Purging the service instance")
			workflowhelpers.AsUser(testSetup.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				testSetup.RegularUserContext().TargetSpace()
				Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})

			By("Ensuring the service instance is gone")
			services = cf.Cf("services").Wait(DEFAULT_TIMEOUT)
			Expect(services).To(Exit(0))
			Expect(services).NotTo(Say(instanceName))
			Expect(services).NotTo(Say(appName))
		})
	})

	Context("for space scoped brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BROKER"),
				assets.NewAssets().ServiceBroker,
				testSetup,
			)
			testSetup.RegularUserContext().TargetSpace()
			broker.Push(config)
			broker.Configure()
			broker.CreateSpaceScoped()
			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVCINS")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes the service instance", func() {
			workflowhelpers.AsUser(testSetup.RegularUserContext(), testSetup.ShortTimeout(), func() {
				By("Having a bound service instance")
				createApp := cf.Cf("push", appName, "--no-start", "-b", config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)
				Expect(createApp).To(Exit(0), "failed creating app")
				app_helpers.SetBackend(appName)
				Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				broker.CreateServiceInstance(instanceName)

				services := cf.Cf("services").Wait(DEFAULT_TIMEOUT)
				Expect(services).To(Exit(0))
				Expect(services).To(Say(instanceName))

				bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
				Expect(bindService).To(Exit(0), "failed binding app to service")

				By("Making the broker unavailable")
				Expect(cf.Cf("delete", broker.Name, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				By("Purging the service instance")
				Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				By("Ensuring the service instance is gone")
				services = cf.Cf("services").Wait(DEFAULT_TIMEOUT)
				Expect(services).To(Exit(0))
				Expect(services).NotTo(Say(instanceName))
				Expect(services).NotTo(Say(appName))
			})
		})
	})
})
