package v3_helpers

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
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
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()

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
