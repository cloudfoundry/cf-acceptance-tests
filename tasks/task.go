package tasks

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"

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

var _ = TasksDescribe("v3 tasks", func() {
	var (
		appName string
		appGuid string
	)

	BeforeEach(func() {
		if !Config.GetIncludeTasks() {
			Skip(skip_messages.SkipTasksMessage)
		}
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push", appName, "--no-start", "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		appGuid = app_helpers.GetAppGuid(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Context("tasks lifecycle", func() {
		It("can successfully create and run a task", func() {
			By("creating the task")
			taskName := "mreow"
			// sleep for enough time to see the task is RUNNING
			sleepTime := math.Min(float64(2), float64(Config.DefaultTimeoutDuration().Seconds()))
			command := fmt.Sprintf("sleep %f", sleepTime)
			lastUsageEventGuid := app_helpers.LastAppUsageEventGuid(TestSetup)
			createCommand := cf.Cf("run-task", appName, command, "--name", taskName).Wait(Config.DefaultTimeoutDuration())
			Expect(createCommand).To(Exit(0))

			taskDetails := getTaskDetails(appName)
			sequenceId := taskDetails[0]
			outputName := taskDetails[1]
			outputState := taskDetails[2]
			ouputCommand := taskDetails[len(taskDetails)-2] + " " + taskDetails[len(taskDetails)-1]

			Expect(ouputCommand).To(Equal(command))
			Expect(outputName).To(Equal(taskName))
			Expect(outputState).To(Equal("RUNNING"))

			taskGuid := getGuid(appGuid, sequenceId)

			By("TASK_STARTED AppUsageEvent")
			usageEvents := app_helpers.UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
			start_event := app_helpers.AppUsageEvent{Entity: app_helpers.Entity{State: "TASK_STARTED", ParentAppGuid: appGuid, ParentAppName: appName, TaskGuid: taskGuid}}
			Expect(app_helpers.UsageEventsInclude(usageEvents, start_event)).To(BeTrue())

			By("successfully running")

			Eventually(func() string {
				taskDetails = getTaskDetails(appName)
				outputName = taskDetails[1]
				outputState = taskDetails[2]
				return outputState
			}, Config.DefaultTimeoutDuration()).Should(Equal("SUCCEEDED"))

			Expect(outputName).To(Equal(taskName))

			By("TASK_STOPPED AppUsageEvent")
			usageEvents = app_helpers.UsageEventsAfterGuid(TestSetup, lastUsageEventGuid)
			stop_event := app_helpers.AppUsageEvent{Entity: app_helpers.Entity{State: "TASK_STOPPED", ParentAppGuid: appGuid, ParentAppName: appName, TaskGuid: taskGuid}}
			Expect(app_helpers.UsageEventsInclude(usageEvents, stop_event)).To(BeTrue())
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
