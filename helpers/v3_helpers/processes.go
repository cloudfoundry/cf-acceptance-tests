package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
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
	bytes := session.Wait().Out.Contents()

	processes := ProcessList{}
	json.Unmarshal(bytes, &processes)

	for i, _ := range processes.Processes {
		processes.Processes[i].Name = appName
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

func GetProcessByGuid(processGuid string) Process {
	processURL := fmt.Sprintf("/v3/processes/%s", processGuid)
	session := cf.Cf("curl", processURL)
	bytes := session.Wait().Out.Contents()

	var process Process
	json.Unmarshal(bytes, &process)

	return process
}
