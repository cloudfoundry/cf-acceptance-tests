package v3

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type Result struct {
	FailureReason string `json:"failure_reason"`
}

type Task struct {
	Guid       string `json:"guid"`
	Command    string `json:"command"`
	Name       string `json:"name"`
	State      string `json:"state"`
	Result     Result `json:"result"`
	SequenceId int    `json:"sequence_id"`
}

type Tasks struct {
	Resources []Task `json:"resources"`
}

func getTaskDetails(appName string) []string {
	listCommand := cf.Cf("tasks", appName).Wait(Config.DefaultTimeoutDuration())
	Expect(listCommand).To(Exit(0))
	listOutput := string(listCommand.Out.Contents())
	lines := strings.Split(listOutput, "\n")
	return strings.Fields(lines[4])
}

func getGuid(appGuid string, sequenceId string) string {
	var tasks Tasks
	readCommand := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/tasks", appGuid), "-X", "GET").Wait(Config.DefaultTimeoutDuration())
	Expect(readCommand).To(Exit(0))
	err := json.Unmarshal(readCommand.Out.Contents(), &tasks)
	Expect(err).NotTo(HaveOccurred())

	var task Task
	for _, task = range tasks.Resources {
		parsedSequenceId, _ := strconv.Atoi(sequenceId)
		if parsedSequenceId == task.SequenceId {
			break
		}
	}
	return task.Guid
}

var _ = V3Describe("v3 tasks", func() {
	var (
		appName                         string
		appGuid                         string
		packageGuid                     string
		spaceGuid                       string
		appCreationEnvironmentVariables string
		token                           string
	)

	BeforeEach(func() {
		if !Config.GetIncludeTasks() {
			Skip(skip_messages.SkipTasksMessage)
		}
		appName = random_name.CATSRandomName("APP")
		spaceGuid = GetSpaceGuidFromName(TestSetup.RegularUserContext().Space)
		appCreationEnvironmentVariables = `"foo"=>"bar"`
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		packageGuid = CreatePackage(appGuid)
		token = GetAuthToken()
		uploadUrl := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)
		UploadPackage(uploadUrl, assets.NewAssets().DoraZip, token)
		WaitForPackageToBeReady(packageGuid)
		dropletGuid := StageBuildpackPackage(packageGuid, "ruby_buildpack")
		WaitForDropletToStage(dropletGuid)
		AssignDropletToApp(appGuid, dropletGuid)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
	})

	Context("tasks lifecycle", func() {
		It("can successfully create and run a task", func() {
			By("creating the task")
			taskName := "mreow"
			command := "ls"
			createCommand := cf.Cf("run-task", appName, command, "--name", taskName).Wait(Config.DefaultTimeoutDuration())
			Expect(createCommand).To(Exit(0))

			taskDetails := getTaskDetails(appName)
			sequenceId := taskDetails[0]
			outputName := taskDetails[1]
			outputState := taskDetails[2]
			ouputCommand := taskDetails[len(taskDetails)-1]

			Expect(ouputCommand).To(Equal(command))
			Expect(outputName).To(Equal(taskName))
			Expect(outputState).To(Equal("RUNNING"))

			taskGuid := getGuid(appGuid, sequenceId)

			By("TASK_STARTED AppUsageEvent")
			usageEvents := LastPageUsageEvents(TestSetup)
			start_event := AppUsageEvent{Entity{State: "TASK_STARTED", ParentAppGuid: appGuid, ParentAppName: appName, TaskGuid: taskGuid}}
			Expect(UsageEventsInclude(usageEvents, start_event)).To(BeTrue())

			By("successfully running")

			Eventually(func() string {
				taskDetails = getTaskDetails(appName)
				outputName = taskDetails[1]
				outputState = taskDetails[2]
				return outputState
			}, Config.DefaultTimeoutDuration()).Should(Equal("SUCCEEDED"))

			Expect(outputName).To(Equal(taskName))

			By("TASK_STOPPED AppUsageEvent")
			usageEvents = LastPageUsageEvents(TestSetup)
			stop_event := AppUsageEvent{Entity{State: "TASK_STOPPED", ParentAppGuid: appGuid, ParentAppName: appName, TaskGuid: taskGuid}}
			Expect(UsageEventsInclude(usageEvents, stop_event)).To(BeTrue())
		})
	})

	Context("When cancelling a task", func() {
		var taskId string
		var taskName string

		BeforeEach(func() {
			command := "sleep 100;"
			taskName = "mreow"
			createCommand := cf.Cf("run-task", appName, command, "--name", taskName).Wait(Config.DefaultTimeoutDuration())
			Expect(createCommand).To(Exit(0))

			taskDetails := getTaskDetails(appName)
			taskId = taskDetails[0]
		})

		It("should show task is in FAILED state", func() {
			terminateCommand := cf.Cf("terminate-task", appName, taskId).Wait(Config.DefaultTimeoutDuration())
			Expect(terminateCommand).To(Exit(0))

			var outputSequenceId, outputName, outputState string
			Eventually(func() string {
				taskDetails := getTaskDetails(appName)
				outputSequenceId = taskDetails[0]
				outputName = taskDetails[1]
				outputState = taskDetails[2]
				return outputState
			}, Config.DefaultTimeoutDuration()).Should(Equal("FAILED"))
			Expect(outputName).To(Equal(taskName))
			taskGuid := getGuid(appGuid, outputSequenceId)

			readCommand := cf.Cf("curl", fmt.Sprintf("/v3/tasks/%s", taskGuid), "-X", "GET").Wait(Config.DefaultTimeoutDuration())
			Expect(readCommand).To(Exit(0))

			var readOutput Task
			err := json.Unmarshal(readCommand.Out.Contents(), &readOutput)
			Expect(err).NotTo(HaveOccurred())
			failureReason := readOutput.Result.FailureReason
			Expect(failureReason).To(Equal("task was cancelled"))
		})
	})
})
