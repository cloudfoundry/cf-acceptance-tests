package services_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gbytes"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
)

var _ = Describe("Purging service offerings", func() {
	var broker ServiceBroker
	var appName, instanceName, asyncInstanceName string

	AfterEach(func() {
		app_helpers.AppReport(broker.Name, DEFAULT_TIMEOUT)
		broker.Destroy()
	})

	Context("for public brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				generator.PrefixedRandomName("ps-"),
				assets.NewAssets().ServiceBroker,
				context,
			)
			broker.Push()
			broker.Configure()
			cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				broker.Create()
				broker.PublicizePlans()
			})
			appName = generator.PrefixedRandomName("CATS-APP-ps-")
			instanceName = generator.PrefixedRandomName("CATS-APP-ps-")
			asyncInstanceName = generator.PrefixedRandomName("CATS-APP-ps-")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes all instances and plans of the service, then removes the service offering", func() {
			By("Having bound service instances")
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

			By("Having async service instances")
			Expect(cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[0].Name, asyncInstanceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("service", asyncInstanceName).Wait(DEFAULT_TIMEOUT)).To(Say("create in progress"))

			By("Making the broker unavailable")
			Expect(cf.Cf("delete", broker.Name, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			By("Purging the service offering")
			cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				Expect(cf.Cf("purge-service-offering", broker.Service.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})

			By("Ensuring service instances are gone")
			services = cf.Cf("services").Wait(DEFAULT_TIMEOUT)
			Expect(services).To(Exit(0))
			Expect(services).NotTo(Say(instanceName))
			Expect(services).NotTo(Say(appName))

			By("Ensuring service offerings are gone")
			marketplace := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
			Expect(marketplace).To(Exit(0))
			Expect(marketplace).NotTo(Say(broker.Service.Name))
		})
	})

	Context("for space scoped brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				generator.PrefixedRandomName("prps-"),
				assets.NewAssets().ServiceBroker,
				context,
			)
			cf.TargetSpace(context.RegularUserContext(), context.ShortTimeout())
			broker.Push()
			broker.Configure()
			broker.CreateSpaceScoped()
			appName = generator.PrefixedRandomName("CATS-APP-prps-")
			instanceName = generator.PrefixedRandomName("CATS-APP-prps-")
			asyncInstanceName = generator.PrefixedRandomName("CATS-APP-prps-")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes all instances and plans of the service, then removes the service offering", func() {
			cf.AsUser(context.RegularUserContext(), context.ShortTimeout(), func() {
				By("Having bound service instances")
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

				By("Having async service instances")
				Expect(cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[0].Name, asyncInstanceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("service", asyncInstanceName).Wait(DEFAULT_TIMEOUT)).To(Say("create in progress"))

				By("Making the broker unavailable")
				Expect(cf.Cf("delete", broker.Name, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				By("Purging the service offering")
				Expect(cf.Cf("purge-service-offering", broker.Service.Name, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				By("Ensuring service instances are gone")
				services = cf.Cf("services").Wait(DEFAULT_TIMEOUT)
				Expect(services).To(Exit(0))
				Expect(services).NotTo(Say(instanceName))
				Expect(services).NotTo(Say(appName))

				By("Ensuring service offerings are gone")
				marketplace := cf.Cf("marketplace").Wait(DEFAULT_TIMEOUT)
				Expect(marketplace).To(Exit(0))
				Expect(marketplace).NotTo(Say(broker.Service.Name))
			})
		})
	})
})
