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

var _ = ServicesDescribe("Purging service offerings", func() {
	var broker ServiceBroker
	var appName, instanceName, asyncInstanceName string

	AfterEach(func() {
		app_helpers.AppReport(broker.Name, DEFAULT_TIMEOUT)
		broker.Destroy()
	})

	Context("for public brokers", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				UserContext,
			)
			broker.Push(Config)
			broker.Configure()
			workflowhelpers.AsUser(UserContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
				broker.Create()
				broker.PublicizePlans()
			})
			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVIN")
			asyncInstanceName = random_name.CATSRandomName("SVIN")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes all instances and plans of the service, then removes the service offering", func() {
			By("Having bound service instances")
			createApp := cf.Cf("push", appName, "--no-start", "-b", Config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.AppsDomain).Wait(DEFAULT_TIMEOUT)
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
			workflowhelpers.AsUser(UserContext.AdminUserContext(), DEFAULT_TIMEOUT, func() {
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
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				UserContext,
			)
			workflowhelpers.TargetSpace(UserContext.RegularUserContext(), UserContext.ShortTimeout())
			broker.Push(Config)
			broker.Configure()
			broker.CreateSpaceScoped()
			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVIN")
			asyncInstanceName = random_name.CATSRandomName("SVIN")
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

		It("removes all instances and plans of the service, then removes the service offering", func() {
			workflowhelpers.AsUser(UserContext.RegularUserContext(), UserContext.ShortTimeout(), func() {
				By("Having bound service instances")
				createApp := cf.Cf("push", appName, "--no-start", "-b", Config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.AppsDomain).Wait(DEFAULT_TIMEOUT)
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
