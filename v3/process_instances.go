package v3

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

type ProcessInstance struct {
	Index int    `json:"index"`
	State string `json:"state"`
	Since int    `json:"since"`
}

type ProcessInstancesResource struct {
	ProcessInstances []ProcessInstance `json:"resources"`
}

type ProcessResource struct {
	ProcessInstances []ProcessInstance `json:"process_instances"`
}

type ProcessesResource struct {
	Processes []ProcessResource `json:"resources"`
}

var _ = V3Describe("process instances", func() {
	var (
		appName    string
		webProcess v3_helpers.Process
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		By("pushing the app with three instances")
		Expect(cf.Cf("push", appName, "-i", "3", "-b", Config.GetRubyBuildpackName(), "-p", assets.NewAssets().DoraZip).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		By("waiting until all instances are running")
		Eventually(func(g Gomega) {
			session := cf.Cf("app", appName).Wait()
			g.Expect(session).Should(Say(`instances:\s+3/3`))
		}).Should(Succeed())

		appGuid := app_helpers.GetAppGuid(appName)
		processes := v3_helpers.GetProcesses(appGuid, appName)
		webProcess = v3_helpers.GetProcessByType(processes, "web")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Context("via different v3 API endpoints", func() {
		It("returns an array of process instances", func() {
			By("/v3/processes/:guid/process_instances")
			{
				resource := ProcessInstancesResource{}
				getResource(fmt.Sprintf("v3/processes/%s/process_instances", webProcess.Guid), &resource)
				Expect(len(resource.ProcessInstances)).To(Equal(3))

				for index, instance := range resource.ProcessInstances {
					checkProcessInstance(instance, index, "RUNNING")
				}
			}

			By("/v3/processes/:guid?embed=process_instances")
			{
				resource := ProcessResource{}
				getResource(fmt.Sprintf("v3/processes/%s?embed=process_instances", webProcess.Guid), &resource)
				Expect(len(resource.ProcessInstances)).To(Equal(3))

				for index, instance := range resource.ProcessInstances {
					checkProcessInstance(instance, index, "RUNNING")
				}
			}

			By("/v3/processes?guids=:guid&embed=process_instances")
			{
				resource := ProcessesResource{}
				getResource(fmt.Sprintf("v3/processes?guids=%s&embed=process_instances", webProcess.Guid), &resource)
				Expect(len(resource.Processes)).To(Equal(1))
				Expect(len(resource.Processes[0].ProcessInstances)).To(Equal(3))

				for index, instance := range resource.Processes[0].ProcessInstances {
					checkProcessInstance(instance, index, "RUNNING")
				}
			}
		})
	})

	Context("when stopping the app", func() {
		It("switches the instances' state from RUNNING to DOWN", func() {
			resource := ProcessInstancesResource{}
			getResource(fmt.Sprintf("v3/processes/%s/process_instances", webProcess.Guid), &resource)
			Expect(len(resource.ProcessInstances)).To(Equal(3))

			for index, instance := range resource.ProcessInstances {
				checkProcessInstance(instance, index, "RUNNING")
			}

			By("stopping the app")
			Expect(cf.Cf("stop", appName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			By("waiting until all instances are stopped")
			Eventually(func() []string {
				getResource(fmt.Sprintf("v3/processes/%s/process_instances", webProcess.Guid), &resource)
				states := []string{}
				for _, instance := range resource.ProcessInstances {
					states = append(states, instance.State)
				}
				return states
			}, V3_PROCESS_TIMEOUT, 1*time.Second).Should(Equal([]string{"DOWN", "DOWN", "DOWN"}))
			Expect(len(resource.ProcessInstances)).To(Equal(3))

			for index, instance := range resource.ProcessInstances {
				checkProcessInstance(instance, index, "DOWN")
			}
		})
	})
})

func getResource(url string, resource any) {
	json.Unmarshal(cf.Cf("curl", url).Wait().Out.Contents(), &resource)
}

func checkProcessInstance(instance ProcessInstance, index int, state string) {
	Expect(instance.Index).To(Equal(index))
	Expect(instance.State).To(Equal(state))
	Expect(instance.Since).To(BeNumerically(">", 0))
}
