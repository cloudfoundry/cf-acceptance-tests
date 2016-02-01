package v3

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("v3 tasks", func() {
	config := helpers.LoadConfig()
	if config.IncludeTasks {
		Context("tasks lifecycle", func() {
			var (
				appName                         string
				appGuid                         string
				packageGuid                     string
				spaceGuid                       string
				appCreationEnvironmentVariables string
			)

			BeforeEach(func() {
				appName = generator.PrefixedRandomName("CATS-APP-")
				spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
				appCreationEnvironmentVariables = `"foo"=>"bar"`
				appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
				packageGuid = CreatePackage(appGuid)
				token := GetAuthToken()
				uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
				UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
				WaitForPackageToBeReady(packageGuid)
				dropletGuid := StagePackage(packageGuid, "{}")
				WaitForDropletToStage(dropletGuid)
				AssignDropletToApp(appGuid, dropletGuid)
			})

			AfterEach(func() {
				DeleteApp(appGuid)
			})

			It("can successfully create and run a task", func() {
				type Task struct {
					Guid    string `json:"guid"`
					Command string `json:"command"`
					Name    string `json:"name"`
					State   string `json:"state"`
				}

				By("create")
				var createOutput Task
				postBody := `{"command": "echo 0", "name": "mreow"}`
				createCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks", appGuid), "-X", "POST", "-d", postBody).Wait(DEFAULT_TIMEOUT)
				Expect(createCommand).To(Exit(0))
				json.Unmarshal(createCommand.Out.Contents(), &createOutput)
				Expect(createOutput.Command).To(Equal("echo 0"))
				Expect(createOutput.Name).To(Equal("mreow"))
				Expect(createOutput.State).To(Equal("RUNNING"))

				var readOutput Task
				Eventually(func() string {
					readCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks/%s", appGuid, createOutput.Guid), "-X", "GET").Wait(DEFAULT_TIMEOUT)
					Expect(readCommand).To(Exit(0))
					json.Unmarshal(readCommand.Out.Contents(), &readOutput)
					return readOutput.State
				}, DEFAULT_TIMEOUT).Should(Equal("SUCCEEDED"))
			})
		})
	}
})
