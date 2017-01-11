package buildpacks

import (
	"errors"
	"os/exec"
)

func GetBuildpacks() (string, error) {
	buildpacks, err := exec.Command("cf", "buildpacks").Output()
	if err != nil {
		return "", errors.New("Error getting buildpack list:" + err.Error())
	}

	return string(buildpacks), nil
}
