package services

import (
	"encoding/json"
	"fmt"
)

type UserProvided struct {
	Label        string   `json:"label"`
	Name         string   `json:"name"`
	Tags         []string `json:"tags"`
	InstanceGUID string   `json:"instance_guid"`
	InstanceName string   `json:"instance_name"`
	BindingGUID  string   `json:"binding_guid"`
	BindingName  string   `json:"binding_name"`
	Credentials  struct {
		Password string `json:"password"`
		Username string `json:"username"`
	} `json:"credentials"`
}

type VCAPServicesFile struct {
	UserProvided []UserProvided `json:"user-provided"`
}

func (v *VCAPServicesFile) ReadFromString(response string) error {
	err := json.Unmarshal([]byte(response), v)

	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return err
	}
	return nil
}
