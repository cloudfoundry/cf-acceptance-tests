package services

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type LastOperation struct {
	State string `json:"state"`
}

type Service struct {
	Name          string        `json:"name"`
	LastOperation LastOperation `json:"last_operation"`
}

type Resource struct {
	Entity Service `json:"entity"`
}

type Response struct {
	Resources []Resource `json:"resources"`
}

var _ = Describe("Service Instance Lifecycle", func() {
	var broker ServiceBroker

	waitForAsyncDeletionToComplete := func(broker ServiceBroker, instanceName string) {
		Eventually(func() string {
			serviceDetails := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			return string(serviceDetails.Out.Contents())
		}, 5*time.Minute, 15*time.Second).Should(ContainSubstring("not found"))
	}

	waitForAsyncOperationToComplete := func(broker ServiceBroker, instanceName string) {
		Eventually(func() string {
			serviceDetails := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(serviceDetails).To(Exit(0), "failed getting service instance details")
			return string(serviceDetails.Out.Contents())
		}, 5*time.Minute, 15*time.Second).Should(ContainSubstring("succeeded"))
	}

	Context("Sync broker", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context)
			broker.Push()
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()
		})

		AfterEach(func() {
			broker.Destroy()
		})

		Context("just service instances", func() {
			It("can create a service instance", func() {
				instanceName := generator.RandomName()
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
				Expect(createService).To(Exit(0))

				serviceInfo := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
				Expect(serviceInfo.Out.Contents()).To(ContainSubstring(fmt.Sprintf("Plan: %s", broker.SyncPlans[0].Name)))
			})

			Context("when there is an existing service instance", func() {
				var instanceName string
				BeforeEach(func() {
					instanceName = generator.RandomName()
					createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
					Expect(createService).To(Exit(0), "failed creating service")
				})

				It("can update a service instance", func() {
					updateService := cf.Cf("update-service", instanceName, "-p", broker.SyncPlans[1].Name).Wait(DEFAULT_TIMEOUT)
					Expect(updateService).To(Exit(0))

					serviceInfo := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
					Expect(serviceInfo.Out.Contents()).To(ContainSubstring(fmt.Sprintf("Plan: %s", broker.SyncPlans[1].Name)))
				})

				It("can delete a service instance", func() {
					deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
					Expect(deleteService).To(Exit(0))

					serviceInfo := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
					Expect(serviceInfo.Out.Contents()).To(ContainSubstring("not found"))
				})
			})
		})

		Context("service instances with an app", func() {
			var instanceName, appName string
			BeforeEach(func() {
				appName = generator.RandomName()
				createApp := cf.Cf("push", appName, "-p", assets.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)
				Expect(createApp).To(Exit(0), "failed creating app")

				checkForEvents(appName, []string{"audit.app.create"})

				instanceName = generator.RandomName()
				createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
				Expect(createService).To(Exit(0), "failed creating service")
			})

			It("can bind service to app and check app env and events", func() {
				bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
				Expect(bindService).To(Exit(0), "failed binding app to service")

				checkForEvents(appName, []string{"audit.app.update"})

				restageApp := cf.Cf("restage", appName).Wait(CF_PUSH_TIMEOUT)
				Expect(restageApp).To(Exit(0), "failed restaging app")

				checkForEvents(appName, []string{"audit.app.restage"})

				appEnv := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
				Expect(appEnv).To(Exit(0), "failed get env for app")
				Expect(appEnv.Out.Contents()).To(ContainSubstring(fmt.Sprintf("credentials")))
			})

			Context("when they are bound", func() {
				BeforeEach(func() {
					bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
					Expect(bindService).To(Exit(0), "failed binding app to service")
				})

				It("can unbind service to app and check app env and events", func() {
					unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
					Expect(unbindService).To(Exit(0), "failed unbinding app to service")

					checkForEvents(appName, []string{"audit.app.update"})

					appEnv := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
					Expect(appEnv).To(Exit(0), "failed get env for app")
					Expect(appEnv.Out.Contents()).ToNot(ContainSubstring(fmt.Sprintf("credentials")))
				})
			})
		})
	})

	Context("Async broker", func() {
		BeforeEach(func() {
			broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context)
			broker.Push()
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()
		})

		AfterEach(func() {
			broker.Destroy()
		})

		It("can create, update, bind, unbind, and delete a service instance", func() {
			appName := generator.RandomName()
			createApp := cf.Cf("push", appName, "-p", assets.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)
			Expect(createApp).To(Exit(0), "failed creating app")

			instanceName := generator.RandomName()
			createService := cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(createService).To(Exit(0))
			Expect(createService.Out.Contents()).To(ContainSubstring("Create in progress."))

			waitForAsyncOperationToComplete(broker, instanceName)

			serviceInfo := cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring(fmt.Sprintf("Plan: %s", broker.AsyncPlans[0].Name)))
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring("Status: create succeeded"))
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring("Message: 100 percent done"))

			updateService := cf.Cf("update-service", instanceName, "-p", broker.AsyncPlans[1].Name).Wait(DEFAULT_TIMEOUT)
			Expect(updateService).To(Exit(0))
			Expect(updateService.Out.Contents()).To(ContainSubstring("Update in progress."))

			waitForAsyncOperationToComplete(broker, instanceName)

			serviceInfo = cf.Cf("service", instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(serviceInfo).To(Exit(0), "failed getting service instance details")
			Expect(serviceInfo.Out.Contents()).To(ContainSubstring(fmt.Sprintf("Plan: %s", broker.AsyncPlans[1].Name)))

			bindService := cf.Cf("bind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(bindService).To(Exit(0), "failed binding app to service")

			checkForEvents(appName, []string{"audit.app.update"})

			restageApp := cf.Cf("restage", appName).Wait(CF_PUSH_TIMEOUT)
			Expect(restageApp).To(Exit(0), "failed restaging app")

			checkForEvents(appName, []string{"audit.app.restage"})

			appEnv := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(appEnv).To(Exit(0), "failed get env for app")
			Expect(appEnv.Out.Contents()).To(ContainSubstring(fmt.Sprintf("credentials")))

			unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(DEFAULT_TIMEOUT)
			Expect(unbindService).To(Exit(0), "failed unbinding app to service")

			checkForEvents(appName, []string{"audit.app.update"})

			appEnv = cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(appEnv).To(Exit(0), "failed get env for app")
			Expect(appEnv.Out.Contents()).ToNot(ContainSubstring(fmt.Sprintf("credentials")))

			deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(DEFAULT_TIMEOUT)
			Expect(deleteService).To(Exit(0), "failed making delete request")
			Expect(deleteService.Out.Contents()).To(ContainSubstring("Delete in progress."))

			waitForAsyncDeletionToComplete(broker, instanceName)
		})
	})
})

func checkForEvents(name string, eventNames []string) {
	events := cf.Cf("events", name).Wait(DEFAULT_TIMEOUT)
	Expect(events).To(Exit(0), fmt.Sprintf("failed getting events for %s", name))

	for _, eventName := range eventNames {
		Expect(events.Out.Contents()).To(ContainSubstring(eventName), "failed to find event")
	}
}
