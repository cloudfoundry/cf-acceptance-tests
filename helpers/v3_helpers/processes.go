package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-acceptance-tests/Godeps/_workspace/src/github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

type ProcessList struct {
	Processes []Process `json:"resources"`
}

type Process struct {
	Guid    string `json:"guid"`
	Type    string `json:"type"`
	Command string `json:"command"`
	Name    string `json:"-"`
}

func GetProcesses(appGuid, appName string) []Process {
	processesURL := fmt.Sprintf("/v3/apps/%s/processes", appGuid)
	session := cf.Cf("curl", processesURL)
	bytes := session.Wait(DEFAULT_TIMEOUT).Out.Contents()

	processes := ProcessList{}
	json.Unmarshal(bytes, &processes)

	for i, process := range processes.Processes {
		processes.Processes[i].Name = fmt.Sprintf("v3-%s-%s", appName, process.Type)
	}

	return processes.Processes
}

func GetProcessByType(processes []Process, processType string) Process {
	for _, process := range processes {
		if process.Type == processType {
			return process
		}
	}
	return Process{}
}
