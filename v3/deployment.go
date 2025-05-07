package v3

import (
	"fmt"
	"sync"
	"time"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = V3Describe("deployment", func() {

	var (
		appName              string
		stopCheckingAppAlive chan<- bool
		appCheckerIsDone     <-chan bool
	)

	const numberOfAppCurlChecks = 10

	BeforeEach(func() {
		if !Config.GetIncludeDeployments() {
			Skip(skip_messages.SkipDeploymentsMessage)
		}
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName, "-i", "3", "-b", Config.GetRubyBuildpackName(), "-p", assets.NewAssets().DoraZip).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("waiting until all instances are running")
		Eventually(func(g Gomega) {
			session := cf.Cf("app", appName).Wait()
			g.Expect(session).Should(Say(`instances:\s+3/3`))
		}).Should(Succeed())
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}).Should(ContainSubstring("Hi, I'm Dora"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("Rolling Deployments", func() {
		BeforeEach(func() {
			stopCheckingAppAlive, appCheckerIsDone = checkAppRemainsAlive(appName)
		})

		AfterEach(func() {
			stopCheckingAppAlive <- true
			<-appCheckerIsDone
		})

		It("deploys an app with no downtime", func() {
			By("Pushing a rolling deployment")
			Expect(cf.Cf("push", appName, "--strategy", "rolling", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).Should(Say("Active deployment with status DEPLOYING"))
				g.Expect(session).Should(Say(`strategy:\s+rolling`))
				g.Expect(session).Should(Exit(0))
			}).Should(Succeed())

			By("Verifying the new rolled out process")
			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).ShouldNot(Say("Active deployment"))
				g.Expect(session).Should(Exit(0))
			}).Should(Succeed())

			checkAppCurlResponse(appName, "Hello from a staticfile", numberOfAppCurlChecks)
		})

		It("can be cancelled and rolls back to the previous app", func() {
			By("Pushing a rolling deployment")
			Expect(cf.Cf("push", appName, "--strategy", "rolling", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).Should(Say("Active deployment with status DEPLOYING"))
				g.Expect(session).Should(Say(`strategy:\s+rolling`))
				g.Expect(session).Should(Exit(0))
			}).Should(Succeed())

			By("Cancelling the deployment")
			Expect(cf.Cf("cancel-deployment", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("Verifying the cancel succeeded and we rolled back to old process")
			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).ShouldNot(Say("Active deployment"))
				g.Expect(session).Should(Exit(0))
			}).Should(Succeed())

			checkAppCurlResponse(appName, "Hi, I'm Dora", numberOfAppCurlChecks)
		})

		Context("max-in-flight", func() {
			It("deploys an app with max_in_flight with a rolling deployment", func() {
				By("Pushing a new rolling deployment with max in flight of 2")
				Expect(cf.Cf("push", appName, "--strategy", "rolling", "--max-in-flight", "2", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				Eventually(func(g Gomega) {
					session := cf.Cf("app", appName).Wait()
					g.Expect(session).Should(Say("Active deployment with status DEPLOYING"))
					g.Expect(session).Should(Say(`strategy:\s+rolling`))
					g.Expect(session).Should(Say(`max-in-flight:\s+2`))
				}).Should(Succeed())

				By("Verifying the new app has rolled out to all instances")
				Eventually(func(g Gomega) {
					session := cf.Cf("app", appName).Wait()
					g.Expect(session).ShouldNot(Say("Active deployment"))
				}).Should(Succeed())

				checkAppCurlResponse(appName, "Hello from a staticfile", numberOfAppCurlChecks)
			})
		})
	})

	Describe("Canary deployments", func() {
		BeforeEach(func() {
			stopCheckingAppAlive, appCheckerIsDone = checkAppRemainsAlive(appName)
		})

		AfterEach(func() {
			stopCheckingAppAlive <- true
			<-appCheckerIsDone
		})

		It("deploys an app, transitions to pause, is continued and then deploys successfully", func() {
			By("Pushing a canary deployment")
			Expect(cf.Cf("push", appName, "--strategy", "canary", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).Should(Say("Active deployment with status PAUSED"))
				g.Expect(session).Should(Say("strategy:        canary"))
			}).Should(Succeed())

			By("Checking that both the canary and original apps exist simultaneously")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a staticfile"))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			By("Continuing the deployment")
			Expect(cf.Cf("continue-deployment", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).ShouldNot(Say("Active deployment"))
			}).Should(Succeed())

			By("Verifying the continue succeeded and we rolled out the new process")

			checkAppCurlResponse(appName, "Hello from a staticfile", numberOfAppCurlChecks)
		})

		It("can be cancelled when paused", func() {
			By("Pushing a canary deployment")
			Expect(cf.Cf("push", appName, "--strategy", "canary", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).Should(Say("Active deployment with status PAUSED"))
				g.Expect(session).Should(Say("strategy:        canary"))
			}).Should(Succeed())

			By("Checking that both the canary and original apps exist simultaneously")
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a staticfile"))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			By("Cancelling the deployment")
			Expect(cf.Cf("cancel-deployment", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func(g Gomega) {
				session := cf.Cf("app", appName).Wait()
				g.Expect(session).ShouldNot(Say("Active deployment"))
			}).Should(Succeed())

			By("Verifying the cancel succeeded and we rolled back to old process")
			checkAppCurlResponse(appName, "Hi, I'm Dora", numberOfAppCurlChecks)
		})

		Context("max-in-flight", func() {
			It("deploys an app with max_in_flight after a canary deployment has been continued", func() {
				By("Pushing a new canary deployment with max in flight of 2")
				Expect(cf.Cf("push", appName, "--strategy", "canary", "--max-in-flight", "2", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				By("Waiting for the a canary deployment to be paused")
				Eventually(func(g Gomega) {
					session := cf.Cf("app", appName).Wait()
					g.Expect(session).Should(Say("Active deployment with status PAUSED"))
					g.Expect(session).Should(Say(`strategy:\s+canary`))
				}).Should(Succeed())

				By("Continuing the deployment")
				Expect(cf.Cf("continue-deployment", appName, "--no-wait").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				Eventually(func(g Gomega) {
					session := cf.Cf("app", appName).Wait()
					g.Expect(session).Should(Say("Active deployment with status DEPLOYING"))
					g.Expect(session).Should(Say(`strategy:\s+canary`))
					g.Expect(session).Should(Say(`max-in-flight:\s+2`))
				}).Should(Succeed())

				Eventually(func(g Gomega) {
					session := cf.Cf("app", appName).Wait()
					g.Expect(session).ShouldNot(Say("Active deployment"))
				}).Should(Succeed())

				By("Verifying the new app has rolled out to all instances")

				checkAppCurlResponse(appName, "Hello from a staticfile", numberOfAppCurlChecks)
			})
		})

		Context("instance-steps", func() {
			BeforeEach(func() {
				Expect(cf.Cf("scale", appName, "-i", "4").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			It("deploys an app, transitions to pause and can be continued multiple times and then deploys successfully", func() {
				By("Pushing a canary deployment")
				Expect(cf.Cf("push", appName, "--strategy", "canary", "--instance-steps=25,50,75", "--no-wait", "-b", Config.GetStaticFileBuildpackName(), "-p", assets.NewAssets().Staticfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				for i := 1; i <= 3; i++ {
					By(fmt.Sprintf("Waiting for the a canary deployment to be paused on step %d", i))
					Eventually(func(g Gomega) {
						session := cf.Cf("app", appName).Wait()
						g.Expect(session).Should(Say("Active deployment with status PAUSED"))
						g.Expect(session).Should(Say("strategy:\\s+canary"))
						g.Expect(session).Should(Say(fmt.Sprintf("canary-steps:\\s+%d/%d", i, 3)))
					}).Should(Succeed())

					By(fmt.Sprintf("Checking that both the canary and original apps exist simultaneously on step %d", i))
					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello from a staticfile"))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hi, I'm Dora"))

					By(fmt.Sprintf("Continuing the deployment on step %d", i))
					Expect(cf.Cf("continue-deployment", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				}

				Eventually(func(g Gomega) {
					session := cf.Cf("app", appName).Wait()
					g.Expect(session).ShouldNot(Say("Active deployment"))
				}).Should(Succeed())

				By("Verifying the continue succeeded and we rolled out the new process")
				checkAppCurlResponse(appName, "Hello from a staticfile", numberOfAppCurlChecks)
			})
		})
	})
})

func checkAppRemainsAlive(appName string) (chan<- bool, <-chan bool) {
	doneChannel := make(chan bool, 1)
	appCheckerIsDone := make(chan bool, 1)
	ticker := time.NewTicker(1 * time.Second)
	tickerChannel := ticker.C

	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-doneChannel:
				ticker.Stop()
				appCheckerIsDone <- true
				return
			case <-tickerChannel:
				Expect(helpers.CurlAppRoot(Config, appName)).ToNot(ContainSubstring("404"))
			}
		}
	}()

	return doneChannel, appCheckerIsDone
}

func checkAppCurlResponse(appName string, expected string, numberOfAppCurlChecks int) {
	var wg sync.WaitGroup
	for range numberOfAppCurlChecks {
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			Expect(helpers.CurlAppRoot(Config, appName)).To(ContainSubstring(expected))
		}()
	}
	wg.Wait()
}
