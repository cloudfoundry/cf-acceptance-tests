package capi_experimental

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
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
		dropletGuid     string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		orgName = TestSetup.RegularUserContext().Org
		spaceGUID = GetSpaceGuidFromName(spaceName)
		By("Creating an App")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		By("Creating a Package")
		packageGUID = CreatePackage(appGUID)
		token = GetAuthToken()
		uploadURL := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGUID)

		By("Uploading a Package")
		UploadPackage(uploadURL, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGUID)

		By("Creating a Build")
		buildGUID := StageBuildpackPackage(packageGUID, Config.GetRubyBuildpackName())
		WaitForBuildToStage(buildGUID)
		dropletGuid = GetDropletFromBuild(buildGUID)
		AssignDropletToApp(appGUID, dropletGuid)

		By("Creating a Route")
		By("Starting an App")
		StartApp(appGUID)
		route = fmt.Sprintf("bar.%s", Config.GetAppsDomain())

		By("Registering a Service Broker")
		broker = NewServiceBroker(
			random_name.CATSRandomName("BRKR"),
			assets.NewAssets().ServiceBroker,
			TestSetup,
		)
		broker.Push(Config)
		broker.Configure()
		broker.Create()
		broker.PublicizePlans()

		By("Creating a Service Instance")
		serviceInstance = random_name.CATSRandomName("SVIN")
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
			manifestToApply  string
			expectedManifest string

			applyEndpoint       string
			getManifestEndpoint string
		)

		BeforeEach(func() {
			applyEndpoint = fmt.Sprintf("/v3/apps/%s/actions/apply_manifest", appGUID)
			getManifestEndpoint = fmt.Sprintf("/v3/apps/%s/manifest", appGUID)
		})

		Describe("routing", func() {
			Context("when routes are specified", func() {
				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  instances: 2
  memory: 300M
  buildpack: ruby_buildpack
  disk_quota: 1024M
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
					expectedManifest = fmt.Sprintf(`
applications:
- name: %s
  stack: cflinuxfs2
  buildpacks:
  - ruby_buildpack
  env:
    foo: qux
    snack: walnuts
  routes:
  - route: %s
  services:
  - %s
  processes:
  - command: bundle exec irb
    disk_quota: 1024M
    health-check-type: process
    instances: 0
    memory: 256M
    type: console
  - command: bundle exec rake
    disk_quota: 1024M
    health-check-type: process
    instances: 0
    memory: 256M
    type: rake
  - command: new-command
    disk_quota: 1024M
    health-check-http-endpoint: /env
    health-check-type: http
    instances: 2
    memory: 300M
    timeout: 75
    type: web
  - command: bundle exec rackup config.ru -p $PORT
    disk_quota: 1024M
    health-check-type: process
    instances: 0
    memory: 256M
    type: worker
`, appName, route, serviceInstance)
				})

				It("successfully completes the job", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
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
						session = cf.Cf("app", appName).Wait(Config.DefaultTimeoutDuration())

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

						session = cf.Cf("curl", "-i", getManifestEndpoint)
						Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
						Expect(session).To(Say("200 OK"))

						session = cf.Cf("curl", getManifestEndpoint)
						Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
						response = session.Out.Contents()
						Expect(string(response)).To(MatchYAML(expectedManifest))
					})
				})
			})

			Context("when specifying no-route", func() {
				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  no-route: true
`, appName)
				})

				It("removes existing routes from the app", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
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

					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  random-route: true
`, appName)
				})

				It("successfully adds a random-route", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
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

		Describe("processes", func() {
			BeforeEach(func() {
				manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    instances: 2
    memory: 300M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, appName)
			})

			Context("when the process exists already", func() {
				It("successfully completes the job", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
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
						Eventually(session).Should(Exit(0))

						processes := GetProcesses(appGUID, appName)
						webProcessWithCommandRedacted := GetProcessByType(processes, "web")
						webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
						Expect(webProcess.Command).To(Equal("new-command"))

						session = cf.Cf("get-health-check", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("health check type:\\s+http"))
						Eventually(session).Should(Say("endpoint \\(for http type\\):\\s+/env"))
						Eventually(session).Should(Exit(0))
					})
				})
			})

			Context("when the process doesn't exist already", func() {
				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: potato
    instances: 2
    memory: 300M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, appName)
				})

				It("creates the process and completes the job", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("v3-app", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("potato:0/2"))
						Eventually(session).Should(Exit(0))

						processes := GetProcesses(appGUID, appName)
						potatoProcessWithCommandRedacted := GetProcessByType(processes, "potato")
						potatoProcess := GetProcessByGuid(potatoProcessWithCommandRedacted.Guid)
						Expect(potatoProcess.Command).To(Equal("new-command"))

						session = cf.Cf("v3-get-health-check", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("potato\\s+http\\s+/env"))
						Eventually(session).Should(Exit(0))
					})
				})
			})

			Context("when setting a new droplet", func() {
				BeforeEach(func() {
					manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: bean
    instances: 2
    memory: 300M
    command: new-command
    health-check-type: http
    health-check-http-endpoint: /env
    timeout: 75
`, appName)
				})

				It("does not remove existing processes", func() {
					session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
					Expect(session.Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
					response := session.Out.Contents()
					Expect(string(response)).To(ContainSubstring("202 Accepted"))

					PollJob(GetJobPath(response))

					workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
						target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
						Expect(target).To(Exit(0), "failed targeting")

						session = cf.Cf("v3-app", appName).Wait(Config.DefaultTimeoutDuration())
						Eventually(session).Should(Say("bean:0/2"))
						Eventually(session).Should(Exit(0))
						AssignDropletToApp(appGUID, dropletGuid)

						processes := GetProcesses(appGUID, appName)
						beanProcessWithCommandRedacted := GetProcessByType(processes, "bean")
						beanProcess := GetProcessByGuid(beanProcessWithCommandRedacted.Guid)
						Expect(beanProcess.Command).To(Equal("new-command"))
					})
				})
			})
		})
	})
})
