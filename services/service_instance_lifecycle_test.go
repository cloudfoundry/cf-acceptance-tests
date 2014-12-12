package services

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Service Instance Lifecycle", func() {
	var broker ServiceBroker

	BeforeEach(func() {
		broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context)
		broker.Plans = append(broker.Plans, Plan{Name: generator.RandomName(), ID: generator.RandomName()})
		broker.Push()
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()
	})

	AfterEach(func() {
		broker.Destroy()
	})

	Context("just service instances", func() {
		It("can create, update, and delete a service instance", func() {
			instanceName := generator.RandomName()
			createService := cf.Cf("create-service", broker.Service.Name, broker.Plans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(createService).To(Exit(0))

			serviceInfo := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring(fmt.Sprintf("Plan: %s", broker.Plans[0].Name)))

			updateService := cf.Cf("update-service", instanceName, "-p", broker.Plans[1].Name).Wait(DEFAULT_TIMEOUT)
			Expect(updateService).To(Exit(0))

			serviceInfo = cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring(fmt.Sprintf("Plan: %s", broker.Plans[1].Name)))

			deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
			Expect(deleteService).To(Exit(0))

			serviceInfo = cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring("not found"))
		})
	})

	Context("service instances with an app", func() {
		var (
			appName, instanceName string
		)

		It("can bind and unbing service to app and check app env", func() {
			appName = generator.RandomName()
			createApp := cf.Cf("push", appName, "-p", assets.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)
			Expect(createApp).To(Exit(0), "failed creating app")

			instanceName = generator.RandomName()
			createService := cf.Cf("create-service", broker.Service.Name, broker.Plans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(createService).To(Exit(0), "failed creating service")

			bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(bindService).To(Exit(0), "failed binding app to service")

			restageApp := cf.Cf("restage", appName).Wait(CF_PUSH_TIMEOUT)
			Expect(restageApp).To(Exit(0), "failed restaging app")

			appEnv := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(appEnv).To(Exit(0), "failed get env for app")
			Expect(appEnv.Out.Contents()).To(ContainSubstring(fmt.Sprintf("credentials")))

			unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(unbindService).To(Exit(0), "failed unbinding app to service")

			appEnv = cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(appEnv).To(Exit(0), "failed get env for app")
			Expect(appEnv.Out.Contents()).ToNot(ContainSubstring(fmt.Sprintf("credentials")))
		})
	})
})
