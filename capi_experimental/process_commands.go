package capi_experimental

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = CapiExperimentalDescribe("setting_process_commands", func() {
	var (
		appName             string
		appGUID             string
		manifestToApply     string
		nullCommandManifest string
		applyEndpoint       string
		packageGUID         string
		spaceGUID           string
		spaceName           string
		token               string
		dropletGuid         string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGUID = GetSpaceGuidFromName(spaceName)
		By("Creating an App")
		appGUID = CreateApp(appName, spaceGUID, `{"foo":"bar"}`)
		applyEndpoint = fmt.Sprintf("/v3/apps/%s/actions/apply_manifest", appGUID)
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

	})

	AfterEach(func() {
		FetchRecentLogs(appGUID, token, Config)
		DeleteApp(appGUID)
	})

	Describe("manifest and Procfile/detected buildpack command interactions", func() {
		manifestToApply = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    command: manifest-command.sh
`, appName)

		nullCommandManifest = fmt.Sprintf(`
applications:
- name: "%s"
  processes:
  - type: web
    command: null
`, appName)

		It("prioritizes the manifest command over the Procfile and can be reset via the API", func() {
			session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
			Expect(session.Wait()).To(Exit(0))
			response := session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("202 Accepted"))

			PollJob(GetJobPath(response))

			processes := GetProcesses(appGUID, appName)
			webProcessWithCommandRedacted := GetProcessByType(processes, "web")
			webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			AssignDropletToApp(appGUID, dropletGuid)

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			processEndpoint := fmt.Sprintf("/v3/processes/%s", webProcessWithCommandRedacted.Guid)
			session = cf.Cf("curl", processEndpoint, "-X", "PATCH", "-d", `{ "command": null }`, "-i")
			Expect(session.Wait()).To(Exit(0))
			response = session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("200 OK"))

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("bundle exec rackup config.ru -p $PORT"))
		})

		It("prioritizes the manifest command over the Procfile and can be reset via manifest", func() {
			session := cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", manifestToApply, "-i")
			Expect(session.Wait()).To(Exit(0))
			response := session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("202 Accepted"))

			PollJob(GetJobPath(response))

			processes := GetProcesses(appGUID, appName)
			webProcessWithCommandRedacted := GetProcessByType(processes, "web")
			webProcess := GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			AssignDropletToApp(appGUID, dropletGuid)

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("manifest-command.sh"))

			session = cf.Cf("curl", applyEndpoint, "-X", "POST", "-H", "Content-Type: application/x-yaml", "-d", nullCommandManifest, "-i")
			Expect(session.Wait()).To(Exit(0))
			response = session.Out.Contents()
			Expect(string(response)).To(ContainSubstring("202 Accepted"))

			PollJob(GetJobPath(response))

			webProcess = GetProcessByGuid(webProcessWithCommandRedacted.Guid)
			Expect(webProcess.Command).To(Equal("bundle exec rackup config.ru -p $PORT"))
		})
	})
})
