package v3

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega"
	. "github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/onsi/gomega/gexec"
)

var _ = Describe("v3 tasks", func() {
	type Result struct {
		FailureReason string `json:"failure_reason"`
	}

	type Task struct {
		Guid    string `json:"guid"`
		Command string `json:"command"`
		Name    string `json:"name"`
		State   string `json:"state"`
		Result  Result `json:"result"`
	}

	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
		token                           string
	)

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		spaceGuid = GetSpaceGuidFromName(context.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo"=>"bar"`
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s/v3/packages/%s/upload", config.ApiEndpoint, packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
		dropletGuid := StageBuildpackPackage(packageGuid, "ruby_buildpack")
		WaitForDropletToStage(dropletGuid)
		AssignDropletToApp(appGuid, dropletGuid)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, config)
		DeleteApp(appGuid)
	})

	config := helpers.LoadConfig()

	if config.IncludeTasks {
		Context("tasks lifecycle", func() {
			It("can successfully create and run a task", func() {
				By("creating the task")
				var createOutput Task
				postBody := `{"command": "echo 0", "name": "mreow"}`
				createCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks", appGuid), "-X", "POST", "-d", postBody).Wait(DEFAULT_TIMEOUT)
				Expect(createCommand).To(Exit(0))
				err := json.Unmarshal(createCommand.Out.Contents(), &createOutput)
				Expect(err).NotTo(HaveOccurred())
				Expect(createOutput.Command).To(Equal("echo 0"))
				Expect(createOutput.Name).To(Equal("mreow"))
				Expect(createOutput.State).To(Equal("RUNNING"))

				By("TASK_STARTED AppUsageEvent")
				usageEvents := LastPageUsageEvents(context)
				start_event := AppUsageEvent{Entity{State: "TASK_STARTED", ParentAppGuid: appGuid, ParentAppName: appName, TaskGuid: createOutput.Guid}}
				Expect(UsageEventsInclude(usageEvents, start_event)).To(BeTrue())

				By("successfully running")
				var readOutput Task
				Eventually(func() string {
					readCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks/%s", appGuid, createOutput.Guid), "-X", "GET").Wait(DEFAULT_TIMEOUT)
					Expect(readCommand).To(Exit(0))
					err := json.Unmarshal(readCommand.Out.Contents(), &readOutput)
					Expect(err).NotTo(HaveOccurred())
					return readOutput.State
				}, DEFAULT_TIMEOUT).Should(Equal("SUCCEEDED"))

				By("TASK_STOPPED AppUsageEvent")
				usageEvents = LastPageUsageEvents(context)
				stop_event := AppUsageEvent{Entity{State: "TASK_STOPPED", ParentAppGuid: appGuid, ParentAppName: appName, TaskGuid: createOutput.Guid}}
				Expect(UsageEventsInclude(usageEvents, stop_event)).To(BeTrue())
			})
		})

		Context("When canceling a task", func() {
			var taskGuid string

			BeforeEach(func() {
				postBody := `{"command": "sleep 100;", "name": "mreow"}`
				createCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks", appGuid), "-X", "POST", "-d", postBody).Wait(DEFAULT_TIMEOUT)
				Expect(createCommand).To(Exit(0))

				var createOutput Task
				err := json.Unmarshal(createCommand.Out.Contents(), &createOutput)
				Expect(err).NotTo(HaveOccurred())
				Expect(createOutput.Guid).NotTo(Equal(""))
				taskGuid = createOutput.Guid
			})

			It("should show task is in FAILED state", func() {
				var failureReason string
				cancelCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks/%s/cancel", appGuid, taskGuid), "-X", "PUT").Wait(DEFAULT_TIMEOUT)
				Expect(cancelCommand).To(Exit(0))

				Eventually(func() string {
					readCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks/%s", appGuid, taskGuid), "-X", "GET").Wait(DEFAULT_TIMEOUT)
					Expect(readCommand).To(Exit(0))

					var readOutput Task
					err := json.Unmarshal(readCommand.Out.Contents(), &readOutput)
					Expect(err).NotTo(HaveOccurred())
					failureReason = readOutput.Result.FailureReason
					return readOutput.State
				}, DEFAULT_TIMEOUT).Should(Equal("FAILED"))
				Expect(failureReason).To(Equal("task was cancelled"))
			})
		})
	}
})
