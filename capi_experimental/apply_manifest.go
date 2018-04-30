package capi_experimental

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = CapiExperimentalDescribe("apply_manifest", func() {
	var (
		appName         string
		appGUID         string
		broker          ServiceBroker
		packageGUID     string
		serviceInstance string
		route           string
		spaceGUID       string
		spaceName       string
		orgName         string
		token           string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		orgName = TestSetup.RegularUserContext().Org
		spaceGUID = GetSpaceGuidFromName(spaceName)
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		packageGUID = CreatePackage(appGUID)
		token = GetAuthToken()
		uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

		UploadPackage(uploadURL, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGUID)

		buildGUID := StageBuildpackPackage(packageGUID, Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGUID)
		dropletGuid := GetDropletFromBuild(buildGUID)
		AssignDropletToApp(appGUID, dropletGuid)

		CreateAndMapRoute(appGUID, spaceName, Config.GetAppsDomain(), appName)
		StartApp(appGUID)
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))

		broker = NewServiceBroker(
			random_name.CATSRandomName("BRKR"),
			assets.NewAssets().ServiceBroker,
			TestSetup,
		)
		broker.Push(Config)
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		serviceInstance = random_name.CATSRandomName("SVIN")
		route = fmt.Sprintf("bar.%s", Config.GetAppsDomain())
		createService := cf.Cf("create-service", broker.Service.Name, broker.SyncPlans[0].Name, serviceInstance).Wait(Config.DefaultTimeoutDuration())
		Expect(createService).To(Exit(0), "failed creating service")
	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, token, Config)
		DeleteApp(appGUID)

		broker.Destroy()
	})

	Describe("Applying manifest to existing app", func() {
		var (
			manifest string
			endpoint string
		)

		BeforeEach(func() {
			endpoint = fmt.Sprintf("/v3/apps/%s/actions/apply_manifest", appGUID)
		})

		Context("when configuring the web process", func() {
			Context("when routes are specified", func() {
				BeforeEach(func() {
					manifest = fmt.Sprintf(`
applications:
- name: "%s"
  instances: 2
  memory: 300M
  buildpack: ruby_buildpack
  stack: cflinuxfs2
  services:
  - %s
  routes:
  - route: %s
  env: { foo: qux, snack: walnuts }
  command: new-command
  health-check-type: http
  health-check-http-endpoint: /env
  timeout: 75
`, appName, serviceInstance, route)
				})

				It("successfully completes the job", func() {
					session := cf.Cf("curl", endpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifest, "-i")
					Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("Showing health"))
						Eventually(session).Should(Say("instances:\\s+.*?\\d+/2"))
						Eventually(session).Should(Say("routes:\\s+(?:%s.%s,\\s+)?%s", appName, Config.GetAppsDomain(), route))
						Eventually(session).Should(Say("stack:\\s+cflinuxfs2"))
						Eventually(session).Should(Say("buildpack:\\s+ruby_buildpack"))
						Eventually(session).Should(Exit(0))

						session = cf.Cf("env", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("foo:\\s+qux"))
						Eventually(session).Should(Say("snack:\\s+walnuts"))
						Eventually(session).Should(Exit(0))

						processes := GetProcesses(appGUID, appName)
						webProcessWithCommandRedacted := GetProcessByType(processes, "web")
						webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
						Expect(webProcess.Command).To(Equal("new-command"))

						session = cf.Cf("get-health-check", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("health check type:\\s+http"))
						Eventually(session).Should(Say("endpoint \\(for http type\\):\\s+/env"))
						Eventually(session).Should(Exit(0))

						session = cf.Cf("service", serviceInstance).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("bound apps:\\s+(?:name\\s+binding name\\s+)?%s", appName))
						Eventually(session).Should(Exit(0))
					})
				})
			})

			Context("when specifying no-route", func() {
				BeforeEach(func() {
					manifest = fmt.Sprintf(`
applications:
- name: "%s"
  no-route: true
`, appName)
				})

				It("removes existing routes from the app", func() {
					session := cf.Cf("curl", endpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifest, "-i")
					Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("Showing health"))
						Eventually(session).Should(Say("routes:\\s*\\n"))
						Eventually(session).Should(Exit(0))
					})
				})
			})

			Context("when random-route is specified", func() {
				BeforeEach(func() {
					UnmapAllRoutes(appGUID)

					manifest = fmt.Sprintf(`
applications:
- name: "%s"
  random-route: true
`, appName)
				})

				It("successfully adds a random-route", func() {
					session := cf.Cf("curl", endpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifest, "-i")
					Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("routes:\\s+%s-\\w+-\\w+.%s", appName, Config.GetAppsDomain()))
					})
				})
			})
		})
	})
})
