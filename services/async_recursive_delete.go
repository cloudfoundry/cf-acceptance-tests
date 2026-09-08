package services_test

import (
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = AsyncRecursiveDeleteDescribe("Async Recursive Delete", func() {
	const asyncOperationPollInterval = 5 * time.Second

	targetOrgAndSpace := func() {
		orgName := TestSetup.RegularUserContext().Org
		spaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0), "failed targeting org and space")
	}

	assertServiceNotFound := func(instanceName string) {
		Eventually(func() *Buffer {
			session := cf.Cf("service", instanceName).Wait()
			combinedOutputBytes := append(session.Out.Contents(), session.Err.Contents()...)
			return BufferWithBytes(combinedOutputBytes)
		}, Config.AsyncServiceOperationTimeoutDuration(), asyncOperationPollInterval).Should(Say("not found"))
	}

	assertAppNotFound := func(appName string) {
		Eventually(func() *Buffer {
			session := cf.Cf("app", appName).Wait()
			combinedOutputBytes := append(session.Out.Contents(), session.Err.Contents()...)
			return BufferWithBytes(combinedOutputBytes)
		}, Config.AsyncServiceOperationTimeoutDuration(), asyncOperationPollInterval).Should(Say("not found"))
	}

	Describe("async unbind via cf delete-service and cf delete", func() {
		var broker ServiceBroker
		var appName, instanceName string

		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			broker.Push(Config)
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()

			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVIN")

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org
				spaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0), "failed targeting org and space")

				Expect(cf.Cf(app_helpers.CatnipWithArgs(
					appName,
					"-m", DEFAULT_MEMORY_LIMIT)...,
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing app")

				createService := cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[2].Name, instanceName).Wait()
				Expect(createService).To(Exit(0), "failed creating async service instance")

				Eventually(func() *Session {
					return cf.Cf("service", instanceName).Wait()
				}, Config.AsyncServiceOperationTimeoutDuration(), asyncOperationPollInterval).Should(Say("succeeded"))

				Expect(cf.Cf("bind-service", appName, instanceName).Wait()).To(Exit(0), "failed binding service to app")

				Eventually(func() *Session {
					return cf.Cf("service", instanceName).Wait()
				}, Config.AsyncServiceOperationTimeoutDuration(), asyncOperationPollInterval).Should(Say(appName + ".*\\ssucceeded"))
			})
		})

		AfterEach(func() {
			app_helpers.AppReport(broker.Name)
			app_helpers.AppReport(appName)

			broker.Destroy()
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org
				spaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()
				cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				cf.Cf("unbind-service", appName, instanceName).Wait()
				cf.Cf("delete-service", instanceName, "-f").Wait()
				cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())
			})
		})

		It("deletes a service instance with an async unbind when using cf delete-service", func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				targetOrgAndSpace()
				deleteService := cf.Cf("delete-service", instanceName, "-f").Wait()
				Expect(deleteService).To(Exit(0), "failed to delete service instance")

				assertServiceNotFound(instanceName)

				getApp := cf.Cf("app", appName).Wait()
				Expect(getApp).To(Exit(0), "app should still exist after delete-service")
			})
		})

		It("triggers async unbind when deleting a bound app with cf delete -f", func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				targetOrgAndSpace()

				Expect(cf.Cf("delete", appName, "-f").Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "cf delete -f should enqueue successfully")

				Eventually(func() *Session {
					return cf.Cf("service", instanceName).Wait()
				}, Config.AsyncServiceOperationTimeoutDuration(), asyncOperationPollInterval).Should(Say("There are no bound apps for this service."))

				assertAppNotFound(appName)
			})
		})
	})

	Describe("synchronous unbind path", func() {
		var broker ServiceBroker
		var appName, instanceName string

		BeforeEach(func() {
			broker = NewServiceBroker(
				random_name.CATSRandomName("BRKR"),
				assets.NewAssets().ServiceBroker,
				TestSetup,
			)
			broker.Push(Config)
			broker.Configure()
			broker.Create()
			broker.PublicizePlans()

			appName = random_name.CATSRandomName("APP")
			instanceName = random_name.CATSRandomName("SVIN")

			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org
				spaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0), "failed targeting org and space")

				Expect(cf.Cf(app_helpers.CatnipWithArgs(
					appName,
					"-m", DEFAULT_MEMORY_LIMIT)...,
				).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing sync-unbind app")

				createService := cf.Cf("create-service", broker.Service.Name, broker.AsyncPlans[0].Name, instanceName).Wait()
				Expect(createService).To(Exit(0), "failed creating service instance with sync-unbind plan")

				Eventually(func() *Session {
					return cf.Cf("service", instanceName).Wait()
				}, Config.AsyncServiceOperationTimeoutDuration(), asyncOperationPollInterval).Should(Say("succeeded"))

				Expect(cf.Cf("bind-service", appName, instanceName).Wait()).To(Exit(0), "failed binding sync-unbind app")
			})
		})

		AfterEach(func() {
			app_helpers.AppReport(broker.Name)
			app_helpers.AppReport(appName)

			broker.Destroy()
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				orgName := TestSetup.RegularUserContext().Org
				spaceName := TestSetup.RegularUserContext().TestSpace.SpaceName()
				cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				cf.Cf("unbind-service", appName, instanceName).Wait()
				cf.Cf("delete-service", instanceName, "-f").Wait()
				cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())
			})
		})

		It("uses synchronous unbind path (broker returns HTTP 200) when deleting a bound app", func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				targetOrgAndSpace()

				deleteApp := cf.Cf("delete", appName, "-f").Wait(Config.CfPushTimeoutDuration())
				Expect(deleteApp).To(Exit(0), "cf delete -f should succeed synchronously when unbind is synchronous")

				assertAppNotFound(appName)
			})
		})
	})
})
